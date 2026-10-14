package client

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/fxamacker/cbor/v2"
)

// SMP (the Simple Management Protocol used by Zephyr's mcumgr) runs over the
// same console as the shell protocol. A packet is an 8 byte header and a CBOR
// body. On a console it travels as base64 text, split into frames that are
// each one line:
//
//	stream = base64( BE16(len(packet)+2) || packet || BE16(crc) )
//	frame  = marker || base64 text || "\n"
//
// The first frame of a packet starts with smpMarkerFirst, the rest with
// smpMarkerCont. This mirrors Zephyr's serial_util.c.

// SMP operations
const (
	smpOpRead     = 0
	smpOpReadRsp  = 1
	smpOpWrite    = 2
	smpOpWriteRsp = 3
)

// smpVersion2 is the header version field for SMP version 2, which reports
// errors as {"err": {"group": g, "rc": n}}.
const smpVersion2 = 1

// SMP groups and command IDs
const (
	smpGroupImage    = 1
	smpIDImageState  = 0
	smpIDImageUpload = 1
)

const smpHeaderLen = 8

// smpMaxFrame is the longest frame Zephyr accepts, marker and newline
// included.
const smpMaxFrame = 127

// smpFrameRaw is how many raw bytes one frame carries. Each frame is encoded
// on its own on a 3 byte boundary, so its text decodes independently, which
// is what the receiver expects.
const smpFrameRaw = (smpMaxFrame - 3) / 4 * 3

// smpMaxPacket bounds the length a received stream may claim, so a corrupt
// length field cannot make the reassembler buffer without limit.
const smpMaxPacket = 4096

var (
	smpMarkerFirst = []byte{0x06, 0x09}
	smpMarkerCont  = []byte{0x04, 0x14}
)

// smpHeader is the 8 byte SMP header.
type smpHeader struct {
	op      uint8
	version uint8
	flags   uint8
	len     uint16
	group   uint16
	seq     uint8
	id      uint8
}

func (h smpHeader) encode() []byte {
	b := make([]byte, smpHeaderLen)
	b[0] = (h.version&0x3)<<3 | h.op&0x7
	b[1] = h.flags
	binary.BigEndian.PutUint16(b[2:], h.len)
	binary.BigEndian.PutUint16(b[4:], h.group)
	b[6] = h.seq
	b[7] = h.id
	return b
}

func decodeSMPHeader(b []byte) (smpHeader, error) {
	if len(b) < smpHeaderLen {
		return smpHeader{}, fmt.Errorf("SMP header is %v bytes, expected %v", len(b), smpHeaderLen)
	}
	return smpHeader{
		op:      b[0] & 0x7,
		version: (b[0] >> 3) & 0x3,
		flags:   b[1],
		len:     binary.BigEndian.Uint16(b[2:]),
		group:   binary.BigEndian.Uint16(b[4:]),
		seq:     b[6],
		id:      b[7],
	}, nil
}

// crc16XModem is CRC-16/XMODEM: polynomial 0x1021, initial value 0, no
// reflection. Zephyr computes it with crc16_itu_t(0, ...).
func crc16XModem(b []byte) uint16 {
	var crc uint16
	for _, c := range b {
		crc ^= uint16(c) << 8
		for range 8 {
			if crc&0x8000 != 0 {
				crc = crc<<1 ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

// smpPacket builds a packet with a CBOR encoded body.
func smpPacket(op uint8, group uint16, id uint8, seq uint8, body any) ([]byte, error) {
	b, err := cbor.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("SMP body: %w", err)
	}
	if len(b) > 0xffff {
		return nil, fmt.Errorf("SMP body is %v bytes, too long", len(b))
	}

	h := smpHeader{
		op:      op,
		version: smpVersion2,
		len:     uint16(len(b)),
		group:   group,
		seq:     seq,
		id:      id,
	}
	return append(h.encode(), b...), nil
}

// smpParsePacket splits a packet into its header and body.
func smpParsePacket(pkt []byte) (smpHeader, []byte, error) {
	h, err := decodeSMPHeader(pkt)
	if err != nil {
		return h, nil, err
	}
	body := pkt[smpHeaderLen:]
	if int(h.len) != len(body) {
		return h, nil, fmt.Errorf("SMP header says %v body bytes, got %v", h.len, len(body))
	}
	return h, body, nil
}

// smpEncodeFrames encodes a packet as console frames, each ending in "\n",
// ready to be written to the port as is.
func smpEncodeFrames(pkt []byte) []byte {
	raw := make([]byte, 0, len(pkt)+4)
	raw = binary.BigEndian.AppendUint16(raw, uint16(len(pkt)+2))
	raw = append(raw, pkt...)
	raw = binary.BigEndian.AppendUint16(raw, crc16XModem(pkt))

	var out bytes.Buffer
	for i := 0; i < len(raw); i += smpFrameRaw {
		end := min(i+smpFrameRaw, len(raw))
		if i == 0 {
			out.Write(smpMarkerFirst)
		} else {
			out.Write(smpMarkerCont)
		}
		out.WriteString(base64.StdEncoding.EncodeToString(raw[i:end]))
		out.WriteByte('\n')
	}
	return out.Bytes()
}

// smpFrameStart returns where an SMP frame starts in a console line, or -1 if
// the line holds none. A frame does not always start the line: the Zephyr
// shell prints its prompt before the response, even with echo and colors off.
// The marker bytes never occur in base64 text, so the first marker found is
// the frame's start.
func smpFrameStart(line []byte) int {
	start := -1
	for _, m := range [][]byte{smpMarkerFirst, smpMarkerCont} {
		if i := bytes.Index(line, m); i >= 0 && (start < 0 || i < start) {
			start = i
		}
	}
	return start
}

// isSMPFrame reports whether a console line holds an SMP frame.
func isSMPFrame(line []byte) bool {
	return smpFrameStart(line) >= 0
}

// smpReassembler collects frames into packets.
type smpReassembler struct {
	buf    []byte
	active bool
}

func (r *smpReassembler) reset() {
	r.buf = r.buf[:0]
	r.active = false
}

// add takes one frame, without its line terminator, and returns the packet
// once the last frame of it has arrived. A nil packet with a nil error means
// more frames are needed.
func (r *smpReassembler) add(frame []byte) ([]byte, error) {
	switch {
	case bytes.HasPrefix(frame, smpMarkerFirst):
		r.reset()
		r.active = true
	case bytes.HasPrefix(frame, smpMarkerCont):
		if !r.active {
			return nil, errors.New("SMP continuation frame without a first frame")
		}
	default:
		return nil, errors.New("not an SMP frame")
	}

	d, err := base64.StdEncoding.DecodeString(string(frame[2:]))
	if err != nil {
		r.reset()
		return nil, fmt.Errorf("SMP frame: %w", err)
	}
	r.buf = append(r.buf, d...)

	if len(r.buf) < 2 {
		return nil, nil
	}

	// the length counts the packet and its CRC
	want := int(binary.BigEndian.Uint16(r.buf))
	if want < smpHeaderLen+2 || want > smpMaxPacket {
		r.reset()
		return nil, fmt.Errorf("SMP stream claims an invalid length of %v", want)
	}
	if len(r.buf)-2 < want {
		return nil, nil
	}

	pkt := append([]byte(nil), r.buf[2:want]...)
	crc := binary.BigEndian.Uint16(r.buf[want:])
	r.reset()

	if crc16XModem(pkt) != crc {
		return nil, errors.New("SMP CRC check failed")
	}
	return pkt, nil
}

// smpRspErr holds the error fields of a response. Version 1 reports a bare
// "rc", version 2 an "err" map; either may appear.
type smpRspErr struct {
	Rc  *int `cbor:"rc,omitempty"`
	Err *struct {
		Group int `cbor:"group"`
		Rc    int `cbor:"rc"`
	} `cbor:"err,omitempty"`
}

// err returns the error a response carries, if any. rc 0 is success.
func (e smpRspErr) err() error {
	if e.Err != nil && e.Err.Rc != 0 {
		return fmt.Errorf("SMP error: group %v rc %v", e.Err.Group, e.Err.Rc)
	}
	if e.Rc != nil && *e.Rc != 0 {
		return fmt.Errorf("SMP error: rc %v", *e.Rc)
	}
	return nil
}

// smpUploadReq is one image upload chunk. len and sha are sent with the
// first chunk only.
type smpUploadReq struct {
	Image int    `cbor:"image"`
	Off   uint64 `cbor:"off"`
	Data  []byte `cbor:"data"`
	Len   uint64 `cbor:"len,omitempty"`
	Sha   []byte `cbor:"sha,omitempty"`
}

// smpUploadRsp gives the offset the MCU expects next.
type smpUploadRsp struct {
	smpRspErr
	Off *uint64 `cbor:"off,omitempty"`
}

// smpImageStateSet marks an image for test (confirm false) or permanent.
type smpImageStateSet struct {
	Hash    []byte `cbor:"hash"`
	Confirm bool   `cbor:"confirm"`
}

// smpImageSlot describes one image slot in an image state response.
type smpImageSlot struct {
	Image     int    `cbor:"image"`
	Slot      int    `cbor:"slot"`
	Version   string `cbor:"version"`
	Hash      []byte `cbor:"hash"`
	Bootable  bool   `cbor:"bootable"`
	Pending   bool   `cbor:"pending"`
	Confirmed bool   `cbor:"confirmed"`
	Active    bool   `cbor:"active"`
	Permanent bool   `cbor:"permanent"`
}

// smpImageStateRsp is the response to an image state read or set.
type smpImageStateRsp struct {
	smpRspErr
	Images []smpImageSlot `cbor:"images,omitempty"`
}

// slot returns the entry for slot n of image 0, or nil.
func (r smpImageStateRsp) slot(n int) *smpImageSlot {
	for i := range r.Images {
		if r.Images[i].Image == 0 && r.Images[i].Slot == n {
			return &r.Images[i]
		}
	}
	return nil
}
