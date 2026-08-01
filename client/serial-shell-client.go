package client

import (
	"bytes"
	"log"
	"time"

	"github.com/simpleiot/simpleiot/data"
)

// defaultShellTimeout is how long the link may be silent before we declare it
// disconnected. An open file descriptor on a USB serial port says nothing
// about whether anything is alive on the other end.
const defaultShellTimeout = 60 * time.Second

// echoExpire bounds how long an un-echoed write stays in the echo cache. This
// is a memory bound, not the correctness mechanism -- correctness comes from
// matching the timestamp exactly.
const echoExpire = time.Minute

// shellConnectedReassert is how often the watchdog republishes an unchanged
// link state. Without it the state is only ever published on a transition, so
// one lost update stays visible until the next transition, which on a healthy
// link may be never.
const shellConnectedReassert = time.Minute

// echoKey identifies a point for echo suppression purposes.
type echoKey struct {
	typ string
	key string
}

// echoEntry is what we last wrote for a point, so the MCU's echo of it can be
// recognized and dropped.
type echoEntry struct {
	data     []byte
	dataType data.PointDataType
	time     time.Time
	written  time.Time
}

// shellHandshake is written on every connect. Echo and colors are turned off
// so the console carries only what the firmware produces, then point
// streaming is enabled and the full cache requested.
//
// The leading newline clears any partial line left in the shell's input
// buffer, which matters when SIOT attaches to a session someone was typing in.
var shellHandshake = []string{
	"",
	"shell echo off",
	"shell colors off",
	"siot stream on",
	"siot dump",
}

// sendShellHandshake writes the connect sequence to the MCU.
func (sd *SerialDevClient) sendShellHandshake() {
	for _, cmd := range shellHandshake {
		if err := sd.writeShellLine(cmd); err != nil {
			log.Printf("Serial %v: error writing handshake: %v", sd.config.Description, err)
			return
		}
		// The shell processes one command at a time. A short gap keeps a
		// burst from overrunning its input buffer on a slow console.
		time.Sleep(20 * time.Millisecond)
	}
}

// writeShellLine writes one command line to the MCU.
func (sd *SerialDevClient) writeShellLine(line string) error {
	if sd.port == nil {
		return nil
	}
	if sd.config.Debug >= 4 && line != "" {
		log.Printf("SER TX (%v): %v", sd.config.Description, line)
	}
	_, err := sd.port.Write([]byte(line + "\r\n"))
	return err
}

// shellTimeout returns the configured silence timeout, or the default.
func (sd *SerialDevClient) shellTimeout() time.Duration {
	if sd.config.Timeout > 0 {
		return time.Duration(sd.config.Timeout) * time.Second
	}
	return defaultShellTimeout
}

// recordShellWrite notes a point written to the MCU so its echo can be
// suppressed. Called for every point sent in shell mode.
func (sd *SerialDevClient) recordShellWrite(p data.Point) {
	if sd.echoCache == nil {
		sd.echoCache = map[echoKey]echoEntry{}
	}
	sd.echoCache[echoKey{p.Type, p.Key}] = echoEntry{
		data:     append([]byte(nil), p.Data...),
		dataType: p.DataType,
		time:     p.Time,
		written:  time.Now(),
	}
}

// isShellEcho reports whether an inbound point is the MCU echoing back a point
// we just wrote.
//
// The MCU's `p` command handler publishes to the same zbus channel the emitter
// subscribes to, so every point SIOT writes comes straight back. Without this
// the two sides would trade the same point forever: a point with no timestamp
// is stamped time.Now() on arrival, so each lap looks newer than the last and
// the store keeps accepting it.
//
// Matching on the timestamp as well as the value is what makes this exact. A
// value-only match within a time window gets both cases wrong: a congested
// link delays the echo past the window, and a genuine MCU-side change back to
// the same value inside the window is wrongly swallowed.
func (sd *SerialDevClient) isShellEcho(p data.Point) bool {
	e, ok := sd.echoCache[echoKey{p.Type, p.Key}]
	if !ok {
		return false
	}

	if e.dataType != p.DataType || !bytes.Equal(e.data, p.Data) {
		// a real change on the MCU, never suppress it
		return false
	}

	// An echo carries the stamp we sent. Compare the instants rather than
	// the formatted strings so this does not depend on two formatters
	// agreeing.
	if !e.time.Equal(p.Time) {
		return false
	}

	delete(sd.echoCache, echoKey{p.Type, p.Key})
	return true
}

// expireEchoCache drops stale entries so a point written just before a
// disconnect does not linger and swallow a legitimate report after reconnect.
func (sd *SerialDevClient) expireEchoCache() {
	for k, e := range sd.echoCache {
		if time.Since(e.written) > echoExpire {
			delete(sd.echoCache, k)
		}
	}
}

// sendPointsToDeviceShell writes points to the MCU as `p` commands, one per
// line. There are no sequence numbers or acks in shell mode.
func (sd *SerialDevClient) sendPointsToDeviceShell(pts data.Points) error {
	if sd.port == nil {
		return nil
	}

	for _, p := range pts {
		if warn := mcuWouldTruncate(p); warn != "" && sd.config.Debug >= 2 {
			log.Printf("Serial %v: MCU will truncate this point: %v",
				sd.config.Description, warn)
		}

		line, err := formatPointWrite(p)
		if err != nil {
			// not fatal for the link; count it and carry on with the rest
			sd.config.ErrorCount++
			log.Printf("Serial %v: cannot format point %v: %v",
				sd.config.Description, p.Type, err)
			continue
		}

		if err := sd.writeShellLine(line); err != nil {
			return err
		}

		sd.recordShellWrite(p)
		sd.config.Tx++
	}

	if sd.config.Tx > 0 {
		err := SendPoints(sd.nc, sd.natsSub,
			data.Points{data.NewPointFloat(data.PointTypeTx, "", float64(sd.config.Tx))},
			false)
		if err != nil {
			return err
		}
	}

	return nil
}

// handleShellLine processes one line received from the MCU console and returns
// any points to publish.
//
// A line that is neither a point nor a log is not an error. The console
// legitimately carries the boot banner, shell command output, and anything
// else the firmware prints.
// It returns points to publish to the node, and admin points (statistics) to
// publish to the client's own node.
func (sd *SerialDevClient) handleShellLine(line string) (data.Points, data.Points) {
	// Console logging happens here, before classification, so lines that
	// fall through to the ignored case still reach the log. Those are
	// exactly the ones worth seeing when something unexpected is on the wire.
	if sd.config.LogConsole {
		log.Printf("MCU %v: %v", sd.config.Description, line)
	}

	p, kind, err := parseShellLine(line)
	if err != nil {
		sd.config.ErrorCount++
		if sd.config.Debug >= 2 {
			log.Printf("Serial %v: %v: %q", sd.config.Description, err, line)
		}
		return nil, data.Points{
			data.NewPointFloat(data.PointTypeErrorCount, "", float64(sd.config.ErrorCount)),
		}
	}

	switch kind {
	case lineLog:
		if sd.config.Debug >= 1 {
			log.Printf("Serial %v: log: %v", sd.config.Description, line)
		}
		return data.Points{data.NewPointString(data.PointTypeLog, "", line)}, nil

	case linePoint:
		if sd.isShellEcho(p) {
			if sd.config.Debug >= 4 {
				log.Printf("SER RX (%v) echo suppressed: %v.%v",
					sd.config.Description, p.Type, p.Key)
			}
			return nil, nil
		}

		if sd.config.Debug >= 4 {
			log.Printf("SER RX (%v): %v", sd.config.Description, line)
		}

		return data.Points{p}, nil
	}

	return nil, nil
}
