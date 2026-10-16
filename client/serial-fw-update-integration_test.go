package client_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
)

// fakeMCU answers the shell handshake with firmware version points and
// implements enough of the SMP image group to accept an upload: it stores
// the chunks, reports image state, and marks an image for test.
type fakeMCU struct {
	rig *shellTestRig
	// running is the fwVersion reported; empty means firmware without
	// update support, which reports nothing
	running string
	staged  string
	tlvHash []byte

	mu           sync.Mutex
	stored       []byte
	total        uint64
	sha          []byte
	uploadStarts int
	smpFrames    int
	marked       bool
	applyLines   []string
	// dropOnce lists upload offsets whose response is lost once
	dropOnce map[uint64]bool
	// revert simulates MCUboot reverting the image: after the mark, the MCU
	// reports it staged but no longer pending
	revert bool

	stop chan struct{}
	done chan struct{}
}

func startFakeMCU(rig *shellTestRig, running, staged string, tlvHash []byte) *fakeMCU {
	f := &fakeMCU{
		rig:      rig,
		running:  running,
		staged:   staged,
		tlvHash:  tlvHash,
		dropOnce: map[uint64]bool{},
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
	go f.run()
	return f
}

func (f *fakeMCU) close() {
	close(f.stop)
	<-f.done
}

func (f *fakeMCU) write(b []byte) {
	if _, err := f.rig.fifo.Write(b); err != nil {
		f.rig.t.Errorf("fake MCU write: %v", err)
	}
}

func (f *fakeMCU) run() {
	defer close(f.done)
	var rx client.SMPReassembler
	for {
		var line string
		select {
		case <-f.stop:
			return
		case line = <-f.rig.lines:
		}

		switch {
		case line == "siot dump" && f.running != "":
			f.write([]byte("pt fwVersion 0 STR " + f.running + "\r\n" +
				"pt fwStagedVersion 0 STR none\r\n" +
				"pt fwPending 0 INT 0\r\n"))
		case strings.HasPrefix(line, "p fwApply "):
			f.mu.Lock()
			f.applyLines = append(f.applyLines, line)
			f.mu.Unlock()
			// the firmware acknowledges every apply request
			f.write([]byte("pt fwApply 0 INT 0\r\n"))
		case client.IsSMPFrame([]byte(line)):
			f.mu.Lock()
			f.smpFrames++
			f.mu.Unlock()
			pkt, err := rx.Add([]byte(line))
			if err != nil {
				f.rig.t.Errorf("fake MCU: %v", err)
				continue
			}
			if pkt != nil {
				f.handle(pkt)
			}
		}
	}
}

func (f *fakeMCU) respond(h client.SMPHeader, body any) {
	h.Op++ // read -> read rsp, write -> write rsp
	pkt, err := client.SMPPacket(h, body)
	if err != nil {
		f.rig.t.Errorf("fake MCU: %v", err)
		return
	}
	// The Zephyr shell prints its prompt ahead of the response.
	f.write(append([]byte("uart:~$ "), client.SMPEncodeFrames(pkt)...))
}

// images returns the image state the MCU reports. Slot 1 is only valid once
// the whole image has arrived intact.
func (f *fakeMCU) images() map[string]any {
	imgs := []any{map[string]any{
		"image": 0, "slot": 0, "version": f.running,
		"hash": bytes.Repeat([]byte{1}, 32), "bootable": true,
		"pending": false, "confirmed": true, "active": true, "permanent": false,
	}}

	sum := sha256.Sum256(f.stored)
	if uint64(len(f.stored)) == f.total && bytes.Equal(sum[:], f.sha) {
		imgs = append(imgs, map[string]any{
			"image": 0, "slot": 1, "version": f.staged, "hash": f.tlvHash,
			"bootable": true, "pending": f.marked, "confirmed": false,
			"active": false, "permanent": false,
		})
	}
	return map[string]any{"images": imgs}
}

func (f *fakeMCU) handle(pkt []byte) {
	h, body, err := client.SMPParse(pkt)
	if err != nil {
		f.rig.t.Errorf("fake MCU: %v", err)
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	switch {
	case h.Group == 1 && h.ID == 1 && h.Op == 2:
		var req struct {
			Off  uint64 `cbor:"off"`
			Data []byte `cbor:"data"`
			Len  uint64 `cbor:"len"`
			Sha  []byte `cbor:"sha"`
		}
		if err := cbor.Unmarshal(body, &req); err != nil {
			f.rig.t.Errorf("fake MCU: upload: %v", err)
			return
		}
		if req.Off == 0 {
			f.uploadStarts++
			f.stored = nil
			f.total = req.Len
			f.sha = req.Sha
			f.marked = false
		}
		// like Zephyr, an unexpected offset is answered with the one wanted
		if req.Off == uint64(len(f.stored)) {
			f.stored = append(f.stored, req.Data...)
			if f.dropOnce[req.Off] {
				delete(f.dropOnce, req.Off)
				return
			}
		}
		f.respond(h, map[string]any{"rc": 0, "off": len(f.stored)})

	case h.Group == 1 && h.ID == 0 && h.Op == 0:
		f.respond(h, f.images())

	case h.Group == 1 && h.ID == 0 && h.Op == 2:
		var req struct {
			Hash    []byte `cbor:"hash"`
			Confirm bool   `cbor:"confirm"`
		}
		if err := cbor.Unmarshal(body, &req); err != nil {
			f.rig.t.Errorf("fake MCU: state: %v", err)
			return
		}
		if req.Confirm {
			f.rig.t.Error("fake MCU: SIOT must never confirm an image")
		}
		if !bytes.Equal(req.Hash, f.tlvHash) {
			f.respond(h, map[string]any{"err": map[string]any{"group": 1, "rc": 2}})
			return
		}
		f.marked = true
		f.respond(h, f.images())
		pending := "1"
		if f.revert {
			pending = "0"
		}
		f.write([]byte("pt fwVersion 0 STR " + f.running + "\r\n" +
			"pt fwStagedVersion 0 STR " + f.staged + "\r\n" +
			"pt fwPending 0 INT " + pending + "\r\n"))

	default:
		f.rig.t.Errorf("fake MCU: unexpected SMP request %+v", h)
	}
}

func (f *fakeMCU) state() (starts, frames int, marked bool, stored []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.uploadStarts, f.smpFrames, f.marked, append([]byte(nil), f.stored...)
}

// writeImage puts a 1.1.0 image in a temporary directory. The payload spans
// several upload chunks.
func writeImage(t *testing.T) (path string, img, tlvHash []byte) {
	t.Helper()
	img, tlvHash = client.BuildMcubootImage(1, 1, 0, 1500)
	path = filepath.Join(t.TempDir(), "mcu.signed.bin")
	if err := os.WriteFile(path, img, 0o644); err != nil {
		t.Fatal(err)
	}
	return path, img, tlvHash
}

func setupFwRig(t *testing.T, running string) (*shellTestRig, *fakeMCU, []byte, func()) {
	t.Helper()
	return setupFwRigWith(t, running, nil)
}

// setupFwRigWith is setupFwRig with extra serial node configuration.
func setupFwRigWith(t *testing.T, running string, cfg func(*client.SerialDev)) (
	*shellTestRig, *fakeMCU, []byte, func(),
) {
	t.Helper()
	path, img, tlvHash := writeImage(t)
	rig, teardown := setupShellRig(t, func(c *client.SerialDev) {
		c.FirmwarePath = path
		if cfg != nil {
			cfg(c)
		}
	})
	f := startFakeMCU(rig, running, "1.1.0", tlvHash)
	return rig, f, img, func() {
		f.close()
		teardown()
	}
}

func TestSerialFwUpdate(t *testing.T) {
	rig, f, img, teardown := setupFwRig(t, "1.0.0")
	defer teardown()

	rig.waitFor("upload to finish", 15*time.Second, func(n client.SerialDev) bool {
		return n.Progress == 100
	})

	starts, _, marked, stored := f.state()
	if !bytes.Equal(stored, img) {
		t.Fatalf("MCU stored %v bytes that do not match the %v byte image", len(stored), len(img))
	}
	if !marked {
		t.Fatal("image was not marked for test")
	}
	if n := rig.getNode(); n.Error != "" {
		t.Errorf("got error %q after a successful update", n.Error)
	}
	rig.waitFor("attempt to be recorded", 3*time.Second, func(n client.SerialDev) bool {
		return n.FwAttemptHash != ""
	})

	// The MCU now reports the image as staged and pending, so nothing
	// further is sent.
	<-time.After(2500 * time.Millisecond)
	if s, _, _, _ := f.state(); s != starts || starts != 1 {
		t.Errorf("got %v upload starts, expected exactly 1", s)
	}

	// Apply now is forwarded with the type the firmware expects, and the
	// MCU's acknowledgment shuts the OS down.
	var poweroffs atomic.Int32
	defer client.SetFwPoweroff(func() error {
		poweroffs.Add(1)
		return nil
	})()
	rig.setPoint(data.PointTypeFwApply, 1)
	start := time.Now()
	for {
		f.mu.Lock()
		lines := append([]string(nil), f.applyLines...)
		f.mu.Unlock()
		if len(lines) > 0 {
			if !strings.HasPrefix(lines[0], "p fwApply 0 INT 1 ") {
				t.Fatalf("got %q, expected an INT fwApply write", lines[0])
			}
			break
		}
		if time.Since(start) > 3*time.Second {
			t.Fatal("timeout waiting for the fwApply write")
		}
		<-time.After(20 * time.Millisecond)
	}

	start = time.Now()
	for poweroffs.Load() == 0 {
		if time.Since(start) > 3*time.Second {
			t.Fatal("timeout waiting for the shutdown after the MCU acknowledged apply")
		}
		<-time.After(20 * time.Millisecond)
	}
	<-time.After(500 * time.Millisecond)
	if n := poweroffs.Load(); n != 1 {
		t.Errorf("got %v shutdowns, expected exactly 1", n)
	}
}

// TestSerialFwUpdateApplyNothingPending checks that Apply now does not shut
// the OS down when the MCU has no image waiting.
func TestSerialFwUpdateApplyNothingPending(t *testing.T) {
	rig, f, _, teardown := setupFwRig(t, "2.0.0")
	defer teardown()

	var poweroffs atomic.Int32
	defer client.SetFwPoweroff(func() error {
		poweroffs.Add(1)
		return nil
	})()

	rig.waitFor("connected", 3*time.Second, func(n client.SerialDev) bool {
		return n.Connected
	})
	<-time.After(1500 * time.Millisecond)

	rig.setPoint(data.PointTypeFwApply, 1)
	start := time.Now()
	for {
		f.mu.Lock()
		n := len(f.applyLines)
		f.mu.Unlock()
		if n > 0 {
			break
		}
		if time.Since(start) > 3*time.Second {
			t.Fatal("timeout waiting for the fwApply write")
		}
		<-time.After(20 * time.Millisecond)
	}
	<-time.After(1 * time.Second)

	if n := poweroffs.Load(); n != 0 {
		t.Errorf("got %v shutdowns with no firmware pending, expected none", n)
	}
}

func TestSerialFwUpdateResume(t *testing.T) {
	rig, f, img, teardown := setupFwRig(t, "1.0.0")
	defer teardown()

	f.mu.Lock()
	f.dropOnce[512] = true
	f.mu.Unlock()

	rig.waitFor("upload to finish", 25*time.Second, func(n client.SerialDev) bool {
		return n.Progress == 100
	})

	starts, _, marked, stored := f.state()
	if starts != 1 {
		t.Errorf("got %v upload starts, expected the upload to resume rather than restart", starts)
	}
	if !bytes.Equal(stored, img) || !marked {
		t.Error("image not stored intact and marked for test")
	}
}

func TestSerialFwUpdateNotNewer(t *testing.T) {
	for _, running := range []string{"1.1.0", "1.2.0", "2.0.0.1"} {
		t.Run(running, func(t *testing.T) {
			rig, f, _, teardown := setupFwRig(t, running)
			defer teardown()

			rig.waitFor("connected", 3*time.Second, func(n client.SerialDev) bool {
				return n.Connected
			})
			<-time.After(3 * time.Second)

			if _, frames, _, _ := f.state(); frames != 0 {
				t.Errorf("got %v SMP frames, expected none when the MCU runs %v", frames, running)
			}
		})
	}
}

func TestSerialFwUpdateNoVersionReported(t *testing.T) {
	rig, f, _, teardown := setupFwRig(t, "")
	defer teardown()

	// old firmware: the link is up but says nothing about its version
	rig.send("pt uptime 0 INT 5")
	rig.waitFor("connected", 3*time.Second, func(n client.SerialDev) bool {
		return n.Connected
	})
	<-time.After(3 * time.Second)

	if _, frames, _, _ := f.state(); frames != 0 {
		t.Errorf("got %v SMP frames, expected none without a fwVersion report", frames)
	}
}

// An image the MCU did not keep is not uploaded again until the file
// changes or fwAttemptHash is cleared.
func TestSerialFwUpdateNoRepeatAfterRevert(t *testing.T) {
	path, _, tlvHash := writeImage(t)
	rig, teardown := setupShellRig(t, func(c *client.SerialDev) {
		c.FirmwarePath = path
	})
	f := &fakeMCU{
		rig: rig, running: "1.0.0", staged: "1.1.0", tlvHash: tlvHash,
		revert: true, dropOnce: map[uint64]bool{},
		stop: make(chan struct{}), done: make(chan struct{}),
	}
	go f.run()
	defer func() {
		f.close()
		teardown()
	}()

	rig.waitFor("not-kept error", 15*time.Second, func(n client.SerialDev) bool {
		return strings.Contains(n.Error, "did not keep firmware 1.1.0 (still running 1.0.0)")
	})

	<-time.After(3 * time.Second)
	if starts, _, marked, _ := f.state(); starts != 1 || !marked {
		t.Errorf("got %v upload starts (marked %v), expected exactly 1", starts, marked)
	}
}

// TestSerialFwUpdateNoRepeatAfterRestart checks that the attempt survives a
// restart: an image MCUboot rejected after an apply, which restarts SIOT, is
// not uploaded and applied again in a loop. Clearing fwAttemptHash retries it.
func TestSerialFwUpdateNoRepeatAfterRestart(t *testing.T) {
	var poweroffs atomic.Int32
	defer client.SetFwPoweroff(func() error {
		poweroffs.Add(1)
		return nil
	})()

	path, _, tlvHash := writeImage(t)
	rig, teardown := setupShellRig(t, func(c *client.SerialDev) {
		c.FirmwarePath = path
		c.FwAutoApply = true
		c.FwAttemptHash = hex.EncodeToString(tlvHash)
	})
	f := startFakeMCU(rig, "1.0.0", "1.1.0", tlvHash)
	defer func() {
		f.close()
		teardown()
	}()

	rig.waitFor("not-kept error", 15*time.Second, func(n client.SerialDev) bool {
		return strings.Contains(n.Error, "did not keep firmware 1.1.0 (still running 1.0.0)")
	})
	<-time.After(2 * time.Second)
	if starts, _, _, _ := f.state(); starts != 0 {
		t.Fatalf("got %v upload starts, expected none for an image already tried", starts)
	}
	if n := poweroffs.Load(); n != 0 {
		t.Fatalf("got %v shutdowns, expected none", n)
	}

	p := data.NewPointString(data.PointTypeFwAttemptHash, "", "")
	p.Origin = "test"
	if err := client.SendNodePoint(rig.nc, rig.nodeID, p, true); err != nil {
		t.Fatal(err)
	}

	rig.waitFor("upload after clearing fwAttemptHash", 15*time.Second,
		func(n client.SerialDev) bool {
			return n.Progress == 100 && n.FwAttemptHash == hex.EncodeToString(tlvHash)
		})
}

// TestSerialFwUpdateAutoApply checks that with fwAutoApply set, a newly
// staged image is applied right away and the system shuts down once.
func TestSerialFwUpdateAutoApply(t *testing.T) {
	var poweroffs atomic.Int32
	defer client.SetFwPoweroff(func() error {
		poweroffs.Add(1)
		return nil
	})()

	rig, f, _, teardown := setupFwRigWith(t, "1.0.0", func(c *client.SerialDev) {
		c.FwAutoApply = true
	})
	defer teardown()

	rig.waitFor("upload to finish", 15*time.Second, func(n client.SerialDev) bool {
		return n.Progress == 100
	})

	start := time.Now()
	for poweroffs.Load() == 0 {
		if time.Since(start) > 5*time.Second {
			t.Fatal("timeout waiting for the automatic apply to shut down")
		}
		<-time.After(20 * time.Millisecond)
	}
	<-time.After(1 * time.Second)

	f.mu.Lock()
	applies := len(f.applyLines)
	f.mu.Unlock()
	if applies != 1 {
		t.Errorf("got %v fwApply writes, expected exactly 1", applies)
	}
	if n := poweroffs.Load(); n != 1 {
		t.Errorf("got %v shutdowns, expected exactly 1", n)
	}
}
