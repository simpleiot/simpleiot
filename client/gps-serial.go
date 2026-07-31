package client

import (
	"bufio"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	nmea "github.com/adrianmo/go-nmea"
	"github.com/fsnotify/fsnotify"
	"github.com/simpleiot/simpleiot/data"
	"go.bug.st/serial"
)

// gpsKnotsToMPS converts knots, the unit NMEA reports speed in, to the
// meters/second SIOT publishes
const gpsKnotsToMPS = 0.514444

// gpsCheckPortDur is how often a missing or failed serial port is retried
const gpsCheckPortDur = 10 * time.Second

// gpsNMEATime builds a UTC timestamp from the separate NMEA date and time
// fields. NMEA carries a two digit year, which is expanded using the usual
// windowing convention.
func gpsNMEATime(d nmea.Date, t nmea.Time) (time.Time, bool) {
	if !d.Valid || !t.Valid {
		return time.Time{}, false
	}

	year := d.YY
	if year >= 80 {
		year += 1900
	} else {
		year += 2000
	}

	return time.Date(year, time.Month(d.MM), d.DD,
		t.Hour, t.Minute, t.Second, t.Millisecond*int(time.Millisecond),
		time.UTC), true
}

// gpsNMEAFixTypes are the sentence types the client consumes. Cycle detection
// only considers these, so a receiver emitting several GSV sentences per cycle
// is not mistaken for the start of a new cycle.
var gpsNMEAFixTypes = map[string]bool{
	nmea.TypeGGA: true,
	nmea.TypeGSA: true,
	nmea.TypeRMC: true,
	nmea.TypeVTG: true,
}

// gpsNMEAAccumulator merges the sentences of one NMEA cycle into a single fix.
//
// A receiver reports one position across several sentences -- GGA carries the
// altitude and fix quality, RMC the speed and heading, GSA the 2D/3D fix type.
// Publishing each sentence separately would emit partial fixes with different
// timestamps, which cannot be joined back into a position. Instead sentences
// accumulate until a type repeats, which marks the start of the next cycle.
type gpsNMEAAccumulator struct {
	parser *nmea.SentenceParser
	// baseValid records whether the sentence passed checksum and structure
	// validation, even if the typed parse then failed
	baseValid bool
	baseType  string

	fix  gpsFix
	seen map[string]bool
}

func newGPSNMEAAccumulator() *gpsNMEAAccumulator {
	a := &gpsNMEAAccumulator{
		parser: &nmea.SentenceParser{},
		seen:   map[string]bool{},
	}

	// The hook fires once the sentence structure and checksum have been
	// validated but before the type-specific fields are parsed. That is what
	// lets add() tell a corrupt sentence apart from a well-formed one whose
	// fields could not be interpreted.
	a.parser.OnBaseSentence = func(s *nmea.BaseSentence) error {
		a.baseValid = true
		a.baseType = s.Type
		return nil
	}

	return a
}

// add parses one NMEA sentence into the accumulator. It returns a completed
// fix when the sentence starts a new cycle, and nil otherwise.
func (a *gpsNMEAAccumulator) add(line string) (*gpsFix, error) {
	a.baseValid = false
	a.baseType = ""

	s, err := a.parser.Parse(strings.TrimSpace(line))

	var typ string
	var apply func()

	switch {
	case err == nil:
		typ = s.DataType()
		apply = func() { a.applySentence(s) }

	case a.baseValid:
		// A well-formed sentence whose fields could not be interpreted. The
		// common case is a GGA whose fix quality falls outside the 0-6 range
		// go-nmea validates against. Reporting no fix is the conservative
		// reading, and more accurate than counting a data error against a
		// receiver that is transmitting correctly.
		typ = a.baseType
		apply = func() { a.applyNoFix(typ) }

	default:
		// failed checksum or malformed structure -- a real data error
		return nil, err
	}

	if !gpsNMEAFixTypes[typ] {
		// a sentence we do not consume, such as GSV
		return nil, nil
	}

	var complete *gpsFix
	if a.seen[typ] {
		// this type has already been seen, so the previous cycle is done
		done := a.fix
		complete = &done
		a.fix = gpsFix{}
		a.seen = map[string]bool{}
	}

	a.seen[typ] = true
	apply()

	return complete, nil
}

// applySentence merges a single parsed sentence into the accumulating fix
func (a *gpsNMEAAccumulator) applySentence(s nmea.Sentence) {
	switch m := s.(type) {
	case nmea.GGA:
		// SIOT's fixQuality adopts the NMEA GGA encoding, so this is a
		// pass-through rather than a mapping. An unparseable quality field is
		// treated as no fix, which is the conservative reading.
		quality := data.PointValueFixQualityNone
		if q, err := strconv.Atoi(m.FixQuality); err == nil {
			quality = q
		}
		a.fix.FixQuality = gpsPtr(quality)

		// satellite count is meaningful even without a fix
		a.fix.NumSat = gpsPtr(int(m.NumSatellites))

		if quality == data.PointValueFixQualityNone {
			// The receiver has no fix. Its GGA leaves the position, altitude,
			// and HDOP fields empty, and go-nmea parses those empty fields as
			// 0 rather than rejecting them -- so without this guard the client
			// would publish 0,0 as a position, which is a real place in the
			// Gulf of Guinea, every time a receiver was searching for
			// satellites.
			return
		}

		a.fix.Latitude = gpsPtr(m.Latitude)
		a.fix.Longitude = gpsPtr(m.Longitude)
		a.fix.Altitude = gpsPtr(m.Altitude)
		a.fix.HDOP = gpsPtr(m.HDOP)

	case nmea.GSA:
		// NMEA GSA numbers a missing fix 1, where SIOT and gpsd both use 0
		switch m.FixType {
		case nmea.FixNone:
			a.fix.FixType = gpsPtr(data.PointValueFixNone)
		case nmea.Fix2D:
			a.fix.FixType = gpsPtr(data.PointValueFix2D)
		case nmea.Fix3D:
			a.fix.FixType = gpsPtr(data.PointValueFix3D)
		}
		if a.fix.HDOP == nil {
			a.fix.HDOP = gpsPtr(m.HDOP)
		}

	case nmea.RMC:
		if m.Validity != nmea.ValidRMC {
			// The receiver is reporting this position is not usable. As with
			// GGA, go-nmea parses the empty position fields as 0, so nothing
			// from this sentence may be trusted.
			//
			// Only report no fix if GGA has not already spoken this cycle.
			// GGA carries the more specific quality field, and letting RMC
			// overwrite it would discard the distinction between a plain GPS
			// fix and an RTK one.
			if a.fix.FixQuality == nil {
				a.fix.FixQuality = gpsPtr(data.PointValueFixQualityNone)
			}
			return
		}
		a.fix.Latitude = gpsPtr(m.Latitude)
		a.fix.Longitude = gpsPtr(m.Longitude)
		a.fix.Speed = gpsPtr(m.Speed * gpsKnotsToMPS)
		a.fix.Heading = gpsPtr(gpsNormalizeHeading(m.Course))
		if t, ok := gpsNMEATime(m.Date, m.Time); ok {
			a.fix.Time = gpsPtr(t)
		}

	case nmea.VTG:
		// only used when the receiver emits no RMC
		if a.fix.Speed == nil {
			a.fix.Speed = gpsPtr(m.GroundSpeedKnots * gpsKnotsToMPS)
		}
		if a.fix.Heading == nil {
			a.fix.Heading = gpsPtr(gpsNormalizeHeading(m.TrueTrack))
		}
	}
}

// applyNoFix records that a sentence reported no usable position
func (a *gpsNMEAAccumulator) applyNoFix(typ string) {
	switch typ {
	case nmea.TypeGGA, nmea.TypeRMC:
		a.fix.FixQuality = gpsPtr(data.PointValueFixQualityNone)
	case nmea.TypeGSA:
		a.fix.FixType = gpsPtr(data.PointValueFixNone)
	}
}

// runSerial reads NMEA sentences from a serial port until stopped
func (gc *GPSClient) runSerial(config GPS, src *gpsSource) {
	conn := &gpsConnState{gc: gc}
	counters := &gpsCounters{gc: gc}

	if config.Port == "" {
		gc.log.Printf("%v: serial port not configured", config.Description)
		conn.set(false)
		<-src.stop
		return
	}

	baud, err := strconv.Atoi(config.Baud)
	if err != nil {
		gc.log.Printf("%v: invalid baud %v", config.Description, config.Baud)
		conn.set(false)
		<-src.stop
		return
	}

	// watch the containing directory so a USB receiver being plugged in or
	// unplugged is noticed without waiting for the retry timer
	var watcherEvents chan fsnotify.Event
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		gc.log.Printf("%v: error creating fsnotify watcher: %v",
			config.Description, err)
	} else {
		defer watcher.Close()
		if err := watcher.Add(filepath.Dir(config.Port)); err != nil {
			gc.log.Printf("%v: error watching %v: %v",
				config.Description, config.Port, err)
		}
		watcherEvents = watcher.Events
	}

	timerCheckPort := time.NewTimer(gpsCheckPortDur)
	defer timerCheckPort.Stop()

	lines := make(chan string, 16)
	readErrors := make(chan error, 4)

	var port serial.Port
	// portQuit is closed to tell the reader goroutine to stop, and readerDone
	// is closed by the reader when it has exited. Both are replaced each time
	// the port is reopened; a nil channel blocks forever in a select, which is
	// the behavior wanted when no reader is running.
	var portQuit chan struct{}
	var readerDone chan struct{}

	closePort := func() {
		if port == nil {
			return
		}
		close(portQuit)
		if err := port.Close(); err != nil {
			gc.log.Printf("%v: error closing port: %v", config.Description, err)
		}
		port = nil
		portQuit = nil
		readerDone = nil
		conn.set(false)
	}
	defer closePort()

	openPort := func() {
		if port != nil {
			return
		}

		p, err := serial.Open(config.Port, &serial.Mode{BaudRate: baud})
		if err != nil {
			if src.debugLevel() >= 2 {
				gc.log.Printf("%v: error opening %v: %v",
					config.Description, config.Port, err)
			}
			timerCheckPort.Reset(gpsCheckPortDur)
			return
		}

		port = p
		portQuit = make(chan struct{})
		readerDone = make(chan struct{})
		gc.log.Printf("%v: serial port opened: %v",
			config.Description, config.Port)

		go gpsSerialReader(p, lines, readErrors, portQuit, readerDone)
	}

	openPort()

	acc := newGPSNMEAAccumulator()

	for {
		select {
		case line := <-lines:
			if src.debugLevel() >= 4 {
				gc.log.Printf("%v: %v", config.Description, line)
			}

			counters.countRx()
			conn.set(true)

			fix, err := acc.add(line)
			if err != nil {
				if src.debugLevel() >= 2 {
					gc.log.Printf("%v: error parsing %q: %v",
						config.Description, line, err)
				}
				counters.countError()
				continue
			}
			if fix != nil {
				gc.publish(*fix)
			}

		case err := <-readErrors:
			if src.debugLevel() >= 2 {
				gc.log.Printf("%v: read error: %v", config.Description, err)
			}
			counters.countError()

		case <-readerDone:
			// the reader hit EOF or a fatal error, usually the receiver being
			// unplugged
			closePort()
			timerCheckPort.Reset(gpsCheckPortDur)

		case <-timerCheckPort.C:
			openPort()

		case e, ok := <-watcherEvents:
			if !ok {
				watcherEvents = nil
				continue
			}
			if e.Name != config.Port {
				continue
			}
			switch {
			case e.Op&fsnotify.Create != 0:
				openPort()
			case e.Op&fsnotify.Remove != 0:
				closePort()
				timerCheckPort.Reset(gpsCheckPortDur)
			}

		case typ := <-src.reset:
			counters.handleReset(typ)

		case <-src.stop:
			return
		}
	}
}

// gpsSerialReader reads NMEA lines from the port and feeds them to the source
// loop. It runs in its own goroutine because serial reads block, and closes
// done when it exits.
func gpsSerialReader(port io.Reader, lines chan<- string,
	readErrors chan<- error, quit <-chan struct{}, done chan struct{}) {

	defer close(done)

	scanner := bufio.NewScanner(port)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		select {
		case lines <- line:
		case <-quit:
			return
		}
	}

	if err := scanner.Err(); err != nil {
		select {
		case readErrors <- err:
		default:
		}
	}
}
