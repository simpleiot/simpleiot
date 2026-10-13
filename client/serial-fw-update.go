package client

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/simpleiot/simpleiot/data"
)

// MCU firmware update over SMP, shell protocol only. The image comes from
// the file named by the firmwarePath point. When it is newer than what the
// MCU reports running, it is uploaded to the MCU's second slot and marked
// for test. The MCU applies it on its own schedule; SIOT never resets it.
//
// Everything here runs on the Run() goroutine, so SMP frames never
// interleave with `p` command writes.

const (
	// fwChunkSize is the image data carried by one upload request.
	fwChunkSize = 512
	// fwRequestTimeout is how long to wait for a response.
	fwRequestTimeout = 5 * time.Second
	// fwFirstChunkTimeout is longer: the MCU may erase the slot before it
	// answers the first chunk.
	fwFirstChunkTimeout = 30 * time.Second
	// fwMaxFailures consecutive timeouts abort an update.
	fwMaxFailures = 3
	// fwRetryBackoff is how long to wait after an aborted update before
	// trying again.
	fwRetryBackoff = 5 * time.Minute
	// fwCheckDelay lets a burst of triggers settle before deciding, such as
	// the three version points the MCU reports together, or a file being
	// written.
	fwCheckDelay = time.Second
)

type fwPhase int

const (
	fwIdle fwPhase = iota
	fwUploading
	fwReadingState
	fwMarkingTest
)

// fwUpdate is the firmware update state of a serial client.
type fwUpdate struct {
	// What the MCU reported since the link came up. A stored fwVersion
	// point from an earlier session says nothing about the firmware running
	// now, so only a fresh report counts.
	reported bool
	version  string
	staged   string
	pending  bool

	phase    fwPhase
	img      *mcubootImage
	off      int // last offset the MCU acknowledged
	seq      uint8
	failures int
	progress int
	rx       smpReassembler

	// err is the error point text the update set, so only that is cleared
	// on success.
	err string
	// retryAt holds off automatic retries after a failed update.
	retryAt time.Time
	// tried is the TLV hash of the last image staged and marked for test.
	// Each image is tried once per process: if the MCU does not keep it,
	// uploading it again would only repeat the same outcome.
	tried []byte

	check   *time.Timer
	timeout *time.Timer
}

func newFwUpdate() fwUpdate {
	f := fwUpdate{
		check:    time.NewTimer(time.Hour),
		timeout:  time.NewTimer(time.Hour),
		progress: -1,
	}
	f.check.Stop()
	f.timeout.Stop()
	return f
}

// fwScheduleCheck evaluates the update decision shortly. During a backoff
// the check waits for the backoff to end.
func (sd *SerialDevClient) fwScheduleCheck() {
	d := fwCheckDelay
	if wait := time.Until(sd.fw.retryAt); wait > d {
		d = wait
	}
	sd.fw.check.Reset(d)
}

// fwConfigChanged is called when the image path or the file itself changes,
// which ends any backoff: a new file deserves a prompt try.
func (sd *SerialDevClient) fwConfigChanged() {
	sd.fw.retryAt = time.Time{}
	sd.fwScheduleCheck()
}

// fwFileEvent reports whether a watcher event is about the image file.
func (sd *SerialDevClient) fwFileEvent(name string) bool {
	return sd.config.FirmwarePath != "" &&
		filepath.Clean(name) == filepath.Clean(sd.config.FirmwarePath)
}

// fwReport records an update point the MCU reported. It returns true if the
// point is one of them.
func (sd *SerialDevClient) fwReport(p data.Point) bool {
	if p.Key != "" {
		return false
	}

	switch p.Type {
	case data.PointTypeFwVersion:
		if sd.fw.reported && sd.fw.version == p.Txt() {
			return true
		}
		sd.fw.reported = true
		sd.fw.version = p.Txt()
	case data.PointTypeFwStagedVersion:
		if sd.fw.staged == p.Txt() {
			return true
		}
		sd.fw.staged = p.Txt()
	case data.PointTypeFwPending:
		pending := p.Val() != 0
		if sd.fw.pending == pending {
			return true
		}
		sd.fw.pending = pending
	default:
		return false
	}

	sd.fwScheduleCheck()
	return true
}

// fwLinkDown forgets what the MCU reported and drops any update in
// progress. The next fwVersion report after the link returns starts over.
func (sd *SerialDevClient) fwLinkDown() {
	if sd.fw.phase != fwIdle {
		log.Printf("Serial %v: MCU firmware update interrupted by link loss",
			sd.config.Description)
		sd.fwStop()
		sd.fwPublishProgress(0)
	}
	sd.fw.reported = false
	sd.fw.version = ""
	sd.fw.staged = ""
	sd.fw.pending = false
	sd.fw.check.Stop()
}

func (sd *SerialDevClient) fwStop() {
	sd.fw.timeout.Stop()
	sd.fw.phase = fwIdle
	sd.fw.img = nil
	sd.fw.rx.reset()
}

func (sd *SerialDevClient) fwSetError(err error) {
	msg := "MCU firmware update: " + err.Error()
	log.Printf("Serial %v: %v", sd.config.Description, msg)
	sd.setError(msg)
	sd.fw.err = msg
}

func (sd *SerialDevClient) fwClearError() {
	if sd.fw.err != "" && sd.config.Error == sd.fw.err {
		sd.setError("")
	}
	sd.fw.err = ""
}

func (sd *SerialDevClient) fwPublishProgress(p int) {
	if p == sd.fw.progress {
		return
	}
	err := SendNodePoint(sd.nc, sd.config.ID,
		data.NewPointFloat(data.PointTypeProgress, "", float64(p)), false)
	if err != nil {
		log.Println("Serial: error sending progress point:", err)
		return
	}
	sd.fw.progress = p
	sd.config.Progress = p
}

// fwCheck decides whether the MCU needs the image and starts the upload.
func (sd *SerialDevClient) fwCheck() {
	if sd.fw.phase != fwIdle || !sd.shell() || sd.port == nil ||
		!sd.config.Connected || sd.config.FirmwarePath == "" || !sd.fw.reported {
		return
	}

	if time.Now().Before(sd.fw.retryAt) {
		sd.fwScheduleCheck()
		return
	}

	path := sd.config.FirmwarePath
	if !filepath.IsAbs(path) {
		sd.fwSetError(fmt.Errorf("firmware path %q must be absolute", path))
		return
	}
	if err := safePath(path); err != nil {
		sd.fwSetError(err)
		return
	}

	img, err := loadMcubootImage(path)
	if err != nil {
		sd.fwSetError(fmt.Errorf("%v: %w", path, err))
		return
	}

	running, err := parseMcubootVersion(sd.fw.version)
	if err != nil {
		sd.fwSetError(fmt.Errorf("MCU reported fwVersion: %w", err))
		return
	}

	if img.version.compare(running) <= 0 {
		sd.fwClearError()
		if sd.config.Debug >= 2 {
			log.Printf("Serial %v: MCU runs firmware %v, image is %v, no update",
				sd.config.Description, running, img.version)
		}
		return
	}

	if sd.fw.pending {
		staged, err := parseMcubootVersion(sd.fw.staged)
		if err == nil && staged.compare(img.version) == 0 {
			sd.fwClearError()
			if sd.config.Debug >= 2 {
				log.Printf("Serial %v: firmware %v already staged on the MCU",
					sd.config.Description, img.version)
			}
			return
		}
	}

	// Staged before, yet the MCU neither runs it nor has it pending: MCUboot
	// reverted it, rejected it, or the mark was lost.
	if bytes.Equal(sd.fw.tried, img.hash) {
		sd.fwSetError(fmt.Errorf("MCU did not keep firmware %v (still running %v); "+
			"not retrying until the file changes or SIOT restarts", img.version, running))
		return
	}

	sd.fwClearError()

	log.Printf("Serial %v: uploading MCU firmware %v (running %v), %v bytes",
		sd.config.Description, img.version, running, len(img.data))

	sd.fw.img = img
	sd.fw.off = 0
	sd.fw.failures = 0
	sd.fw.phase = fwUploading
	sd.fw.rx.reset()
	sd.fwPublishProgress(0)
	sd.fwSendRequest()
}

// fwSendRequest sends the request for the current phase. It is also how a
// timed out request is retried: an upload resumes from the last offset the
// MCU acknowledged.
func (sd *SerialDevClient) fwSendRequest() {
	var (
		op      uint8
		id      uint8
		body    any
		timeout = fwRequestTimeout
	)

	switch sd.fw.phase {
	case fwUploading:
		img := sd.fw.img
		end := min(sd.fw.off+fwChunkSize, len(img.data))
		req := smpUploadReq{
			Off:  uint64(sd.fw.off),
			Data: img.data[sd.fw.off:end],
		}
		if sd.fw.off == 0 {
			req.Len = uint64(len(img.data))
			req.Sha = img.sha[:]
			timeout = fwFirstChunkTimeout
		}
		op, id, body = smpOpWrite, smpIDImageUpload, req
	case fwReadingState:
		op, id, body = smpOpRead, smpIDImageState, map[string]any{}
	case fwMarkingTest:
		op, id, body = smpOpWrite, smpIDImageState,
			smpImageStateSet{Hash: sd.fw.img.hash, Confirm: false}
	default:
		return
	}

	sd.fw.seq++
	pkt, err := smpPacket(op, smpGroupImage, id, sd.fw.seq, body)
	if err != nil {
		sd.fwAbort(err)
		return
	}

	frames := smpEncodeFrames(pkt)
	if sd.config.Debug >= 4 {
		log.Printf("SER TX (%v) SMP seq:%v id:%v %v bytes",
			sd.config.Description, sd.fw.seq, id, len(pkt))
	}

	// the frames carry their own newlines, so they are written as is
	if _, err := sd.port.Write(frames); err != nil {
		sd.fwAbort(fmt.Errorf("writing to the MCU: %w", err))
		return
	}

	sd.fw.timeout.Reset(timeout)
}

// fwRetry handles a request that failed in a way a resend may fix.
func (sd *SerialDevClient) fwRetry(reason error) {
	sd.fw.failures++
	if sd.fw.failures >= fwMaxFailures {
		sd.fwAbort(fmt.Errorf("giving up after %v attempts: %w", sd.fw.failures, reason))
		return
	}
	log.Printf("Serial %v: MCU firmware update: %v, retrying",
		sd.config.Description, reason)
	sd.fw.rx.reset()
	sd.fwSendRequest()
}

// fwTimeout is called when a request gets no response in time.
func (sd *SerialDevClient) fwTimeout() {
	if sd.fw.phase == fwIdle {
		return
	}
	sd.fwRetry(errors.New("no response from the MCU"))
}

// fwAbort ends an update, reports why, and holds off retrying.
func (sd *SerialDevClient) fwAbort(err error) {
	sd.fwStop()
	sd.fwSetError(err)
	sd.fwPublishProgress(0)
	sd.fw.retryAt = time.Now().Add(fwRetryBackoff)
	sd.fwScheduleCheck()
}

// fwHandleFrame processes one SMP frame line from the MCU.
func (sd *SerialDevClient) fwHandleFrame(line []byte) {
	pkt, err := sd.fw.rx.add(line)
	if err != nil {
		if sd.config.Debug >= 2 {
			log.Printf("Serial %v: %v", sd.config.Description, err)
		}
		return
	}
	if pkt == nil {
		return
	}

	h, body, err := smpParsePacket(pkt)
	if err != nil {
		if sd.config.Debug >= 2 {
			log.Printf("Serial %v: %v", sd.config.Description, err)
		}
		return
	}

	if sd.config.Debug >= 4 {
		log.Printf("SER RX (%v) SMP seq:%v group:%v id:%v op:%v",
			sd.config.Description, h.seq, h.group, h.id, h.op)
	}

	if sd.fw.phase == fwIdle || h.seq != sd.fw.seq || h.group != smpGroupImage ||
		(h.op != smpOpReadRsp && h.op != smpOpWriteRsp) {
		// a late answer to a request already retried, or not ours
		return
	}

	sd.fw.timeout.Stop()

	switch sd.fw.phase {
	case fwUploading:
		sd.fwUploadRsp(body)
	case fwReadingState:
		sd.fwStateReadRsp(body)
	case fwMarkingTest:
		sd.fwMarkTestRsp(body)
	}
}

func (sd *SerialDevClient) fwUploadRsp(body []byte) {
	var rsp smpUploadRsp
	if err := cbor.Unmarshal(body, &rsp); err != nil {
		sd.fwRetry(fmt.Errorf("decoding upload response: %w", err))
		return
	}
	if err := rsp.err(); err != nil {
		sd.fwAbort(fmt.Errorf("MCU refused the upload: %w", err))
		return
	}
	if rsp.Off == nil {
		sd.fwRetry(errors.New("upload response has no offset"))
		return
	}

	size := len(sd.fw.img.data)
	if *rsp.Off > uint64(size) {
		sd.fwAbort(fmt.Errorf("MCU acknowledged offset %v of a %v byte image", *rsp.Off, size))
		return
	}

	sd.fw.off = int(*rsp.Off)
	sd.fw.failures = 0
	sd.fwPublishProgress(sd.fw.off * 100 / size)

	if sd.fw.off == size {
		sd.fw.phase = fwReadingState
	}
	sd.fwSendRequest()
}

func (sd *SerialDevClient) fwStateReadRsp(body []byte) {
	var rsp smpImageStateRsp
	if err := cbor.Unmarshal(body, &rsp); err != nil {
		sd.fwRetry(fmt.Errorf("decoding image state: %w", err))
		return
	}
	if err := rsp.err(); err != nil {
		sd.fwAbort(fmt.Errorf("reading image state: %w", err))
		return
	}

	s := rsp.slot(1)
	if s == nil {
		sd.fwAbort(errors.New("MCU reports no image in slot 1 after the upload"))
		return
	}
	if !bytes.Equal(s.Hash, sd.fw.img.hash) {
		sd.fwAbort(errors.New("image in slot 1 does not match the uploaded file"))
		return
	}

	sd.fw.failures = 0
	sd.fw.phase = fwMarkingTest
	sd.fwSendRequest()
}

func (sd *SerialDevClient) fwMarkTestRsp(body []byte) {
	var rsp smpImageStateRsp
	if err := cbor.Unmarshal(body, &rsp); err != nil {
		sd.fwRetry(fmt.Errorf("decoding image state: %w", err))
		return
	}
	if err := rsp.err(); err != nil {
		sd.fwAbort(fmt.Errorf("marking the image for test: %w", err))
		return
	}

	s := rsp.slot(1)
	if s == nil || !s.Pending || !bytes.Equal(s.Hash, sd.fw.img.hash) {
		sd.fwAbort(errors.New("MCU did not mark the image pending"))
		return
	}

	v := sd.fw.img.version.String()
	log.Printf("Serial %v: MCU firmware %v uploaded and marked for test; the MCU applies it on its own schedule",
		sd.config.Description, v)

	// the MCU reports these too; recording them now keeps a check that runs
	// before its report from uploading the same image again
	sd.fw.staged = v
	sd.fw.pending = true
	sd.fw.tried = sd.fw.img.hash

	sd.fwStop()
	sd.fw.retryAt = time.Time{}
	sd.fwClearError()
	sd.fwPublishProgress(100)

	// the file may have changed while the upload ran
	sd.fwScheduleCheck()
}

// fwApply asks the MCU to apply a pending image now.
func (sd *SerialDevClient) fwApply() {
	if !sd.shell() || sd.port == nil {
		return
	}

	log.Printf("Serial %v: asking the MCU to apply its pending firmware", sd.config.Description)

	p := data.NewPointInt(data.PointTypeFwApply, "", 1)
	p.Time = time.Now()
	if err := sd.sendPointsToDeviceShell(data.Points{p}); err != nil {
		log.Println("Serial: error sending fwApply:", err)
	}

	// clear the request on the node, as the counter resets do
	err := SendNodePoint(sd.nc, sd.config.ID,
		data.NewPointFloat(data.PointTypeFwApply, "", 0), false)
	if err != nil {
		log.Println("Serial: error clearing fwApply:", err)
	}
}
