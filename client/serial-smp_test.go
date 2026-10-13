package client

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"testing"

	"github.com/fxamacker/cbor/v2"
)

func TestCRC16XModem(t *testing.T) {
	if got := crc16XModem([]byte("123456789")); got != 0x31C3 {
		t.Fatalf("got 0x%04X, expected 0x31C3", got)
	}
}

func TestSMPHeaderRoundTrip(t *testing.T) {
	h := smpHeader{
		op:      smpOpWriteRsp,
		version: smpVersion2,
		len:     0x1234,
		group:   smpGroupImage,
		seq:     200,
		id:      smpIDImageUpload,
	}

	b := h.encode()
	exp := []byte{0x0b, 0x00, 0x12, 0x34, 0x00, 0x01, 200, 0x01}
	if !bytes.Equal(b, exp) {
		t.Fatalf("encoded % x, expected % x", b, exp)
	}

	got, err := decodeSMPHeader(b)
	if err != nil {
		t.Fatal(err)
	}
	if got != h {
		t.Fatalf("got %+v, expected %+v", got, h)
	}

	if _, err := decodeSMPHeader(b[:7]); err == nil {
		t.Fatal("expected an error for a short header")
	}
}

// splitFrames splits encoded output into frames without terminators.
func splitFrames(t *testing.T, enc []byte) [][]byte {
	t.Helper()
	if !bytes.HasSuffix(enc, []byte("\n")) {
		t.Fatal("encoded frames do not end in a newline")
	}
	return bytes.Split(enc[:len(enc)-1], []byte("\n"))
}

func TestSMPFrameRoundTrip(t *testing.T) {
	// The stream is the packet plus 4 bytes (length and CRC), sent 93 raw
	// bytes per frame. These sizes put frame boundaries around the CRC and
	// the end of the stream.
	sizes := []int{
		smpHeaderLen, 50,
		smpFrameRaw - 4, smpFrameRaw - 3, smpFrameRaw - 2, smpFrameRaw - 1,
		smpFrameRaw, smpFrameRaw + 1,
		2*smpFrameRaw - 4, 2*smpFrameRaw - 3, 2*smpFrameRaw - 2,
		2*smpFrameRaw - 1, 2 * smpFrameRaw,
		600, 1000,
	}

	for _, size := range sizes {
		pkt := make([]byte, size)
		for i := range pkt {
			pkt[i] = byte(i * 7)
		}

		enc := smpEncodeFrames(pkt)
		frames := splitFrames(t, enc)

		expFrames := (size + 4 + smpFrameRaw - 1) / smpFrameRaw
		if len(frames) != expFrames {
			t.Errorf("size %v: got %v frames, expected %v", size, len(frames), expFrames)
		}

		var r smpReassembler
		var got []byte
		for i, f := range frames {
			if len(f)+1 > smpMaxFrame {
				t.Errorf("size %v: frame %v is %v bytes", size, i, len(f)+1)
			}
			marker := smpMarkerCont
			if i == 0 {
				marker = smpMarkerFirst
			}
			if !bytes.HasPrefix(f, marker) || !isSMPFrame(f) {
				t.Fatalf("size %v: frame %v has the wrong marker", size, i)
			}
			// each frame's text must decode on its own
			if _, err := base64.StdEncoding.DecodeString(string(f[2:])); err != nil {
				t.Fatalf("size %v: frame %v does not decode alone: %v", size, i, err)
			}

			p, err := r.add(f)
			if err != nil {
				t.Fatalf("size %v: frame %v: %v", size, i, err)
			}
			if p != nil && i != len(frames)-1 {
				t.Fatalf("size %v: packet complete at frame %v of %v", size, i, len(frames))
			}
			got = p
		}

		if !bytes.Equal(got, pkt) {
			t.Errorf("size %v: round trip mismatch", size)
		}
	}
}

func TestSMPReassemblerErrors(t *testing.T) {
	pkt := make([]byte, 20)
	frames := splitFrames(t, smpEncodeFrames(pkt))

	var r smpReassembler

	if _, err := r.add(append([]byte{0x04, 0x14}, frames[0][2:]...)); err == nil {
		t.Error("expected an error for a continuation with no first frame")
	}

	// flip a bit in the body: the CRC must catch it
	raw, _ := base64.StdEncoding.DecodeString(string(frames[0][2:]))
	raw[5] ^= 0x01
	bad := append(append([]byte{}, smpMarkerFirst...), base64.StdEncoding.EncodeToString(raw)...)
	if _, err := r.add(bad); err == nil {
		t.Error("expected a CRC error")
	}

	if _, err := r.add(append(append([]byte{}, smpMarkerFirst...), "@@@@"...)); err == nil {
		t.Error("expected a base64 error")
	}

	huge := binary.BigEndian.AppendUint16(nil, smpMaxPacket+1)
	huge = append(huge, 0)
	if _, err := r.add(append(append([]byte{}, smpMarkerFirst...),
		base64.StdEncoding.EncodeToString(huge)...)); err == nil {
		t.Error("expected an error for an oversized length")
	}

	// a first frame restarts reassembly, so a good packet follows a bad one
	got, err := r.add(frames[0])
	if err != nil || !bytes.Equal(got, pkt) {
		t.Errorf("got %v, %v after errors, expected the packet", got, err)
	}
}

func TestSMPPacketShapes(t *testing.T) {
	sha := bytes.Repeat([]byte{0xaa}, 32)
	pkt, err := smpPacket(smpOpWrite, smpGroupImage, smpIDImageUpload, 9,
		smpUploadReq{Off: 0, Data: []byte{1, 2, 3}, Len: 1000, Sha: sha})
	if err != nil {
		t.Fatal(err)
	}

	h, body, err := smpParsePacket(pkt)
	if err != nil {
		t.Fatal(err)
	}
	if h.op != smpOpWrite || h.version != smpVersion2 || h.group != smpGroupImage ||
		h.id != smpIDImageUpload || h.seq != 9 {
		t.Fatalf("unexpected header %+v", h)
	}

	var m map[string]any
	if err := cbor.Unmarshal(body, &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"image", "off", "data", "len", "sha"} {
		if _, ok := m[k]; !ok {
			t.Errorf("first chunk is missing %q", k)
		}
	}

	// later chunks carry no len or sha, but always an offset
	pkt, _ = smpPacket(smpOpWrite, smpGroupImage, smpIDImageUpload, 10,
		smpUploadReq{Off: 512, Data: []byte{4}})
	_, body, _ = smpParsePacket(pkt)
	m = nil
	if err := cbor.Unmarshal(body, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["len"]; ok {
		t.Error("later chunk carries len")
	}
	if _, ok := m["sha"]; ok {
		t.Error("later chunk carries sha")
	}
	if m["off"] != uint64(512) {
		t.Errorf("got off %v, expected 512", m["off"])
	}

	// mark for test
	pkt, _ = smpPacket(smpOpWrite, smpGroupImage, smpIDImageState, 11,
		smpImageStateSet{Hash: sha})
	_, body, _ = smpParsePacket(pkt)
	m = nil
	if err := cbor.Unmarshal(body, &m); err != nil {
		t.Fatal(err)
	}
	if m["confirm"] != false || !bytes.Equal(m["hash"].([]byte), sha) {
		t.Errorf("unexpected image state set body %v", m)
	}

	// a read of image state is an empty map
	pkt, _ = smpPacket(smpOpRead, smpGroupImage, smpIDImageState, 12, map[string]any{})
	if _, body, _ = smpParsePacket(pkt); !bytes.Equal(body, []byte{0xa0}) {
		t.Errorf("got body % x, expected an empty map", body)
	}

	// a header length that disagrees with the body is refused
	if _, _, err := smpParsePacket(append(pkt, 0)); err == nil {
		t.Error("expected an error for a length mismatch")
	}
}

func TestSMPResponses(t *testing.T) {
	enc := func(v any) []byte {
		b, err := cbor.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}

	var up smpUploadRsp
	if err := cbor.Unmarshal(enc(map[string]any{"rc": 0, "off": 1024}), &up); err != nil {
		t.Fatal(err)
	}
	if up.err() != nil || up.Off == nil || *up.Off != 1024 {
		t.Errorf("v1 success: got %+v", up)
	}

	up = smpUploadRsp{}
	_ = cbor.Unmarshal(enc(map[string]any{"off": 512}), &up)
	if up.err() != nil || *up.Off != 512 {
		t.Errorf("v2 success: got %+v", up)
	}

	up = smpUploadRsp{}
	_ = cbor.Unmarshal(enc(map[string]any{"rc": 3}), &up)
	if up.err() == nil {
		t.Error("v1 error not reported")
	}

	up = smpUploadRsp{}
	_ = cbor.Unmarshal(enc(map[string]any{"err": map[string]any{"group": 1, "rc": 5}}), &up)
	if up.err() == nil {
		t.Error("v2 error not reported")
	}

	hash := bytes.Repeat([]byte{0x11}, 32)
	var st smpImageStateRsp
	err := cbor.Unmarshal(enc(map[string]any{
		"images": []any{
			map[string]any{"image": 0, "slot": 0, "version": "1.0.0", "hash": []byte{1},
				"bootable": true, "active": true, "confirmed": true},
			map[string]any{"image": 0, "slot": 1, "version": "1.1.0", "hash": hash,
				"bootable": true, "pending": true},
		},
		"splitStatus": 0,
	}), &st)
	if err != nil {
		t.Fatal(err)
	}
	s := st.slot(1)
	if s == nil || !s.Pending || s.Version != "1.1.0" || !bytes.Equal(s.Hash, hash) {
		t.Errorf("slot 1: got %+v", s)
	}
	if st.slot(2) != nil {
		t.Error("found a slot that is not there")
	}
}

// TestSMPFrameAfterPrompt uses a response captured from Zephyr's mcumgr shell
// transport, where the frame follows the shell prompt on the same line.
func TestSMPFrameAfterPrompt(t *testing.T) {
	line := []byte("\x1b[1;32muart:~$ \x1b[m\x06\x09ABQLAAAKAAABAL9hcmVoZWxsb//0JA==")

	i := smpFrameStart(line)
	if i < 0 {
		t.Fatal("frame after the prompt not found")
	}

	var r smpReassembler
	pkt, err := r.add(line[i:])
	if err != nil {
		t.Fatal(err)
	}
	if pkt == nil {
		t.Fatal("expected a complete packet")
	}
	h, _, err := smpParsePacket(pkt)
	if err != nil {
		t.Fatal(err)
	}
	if h.op != smpOpWriteRsp || h.group != 0 || h.id != 0 {
		t.Errorf("got op %v group %v id %v, expected an echo write response", h.op, h.group, h.id)
	}

	if smpFrameStart([]byte("pt fwVersion 0 STR 1.0.0")) >= 0 {
		t.Error("point line taken for an SMP frame")
	}
}
