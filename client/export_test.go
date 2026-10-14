package client

import "github.com/simpleiot/simpleiot/data"

// ScanNodes exposes scanHelper to the external tests.
func (m *Manager[T]) ScanNodes(id string) ([]data.NodeEdge, error) {
	return m.scanHelper(id, nil)
}

// BuildMcubootImage exposes buildMcubootImage to the external tests. It
// returns the image file and its SHA-256 TLV.
func BuildMcubootImage(major, minor uint8, revision uint16, payloadLen int) ([]byte, []byte) {
	return buildMcubootImage(mcubootVersion{major: major, minor: minor, revision: revision},
		payloadLen, true)
}

// SMPEncodeFrames exposes smpEncodeFrames to the external tests.
func SMPEncodeFrames(pkt []byte) []byte {
	return smpEncodeFrames(pkt)
}

// IsSMPFrame exposes isSMPFrame to the external tests.
func IsSMPFrame(line []byte) bool {
	return isSMPFrame(line)
}

// SMPReassembler exposes smpReassembler to the external tests.
type SMPReassembler struct {
	r smpReassembler
}

// Add takes one frame and returns the packet once it is complete.
func (s *SMPReassembler) Add(frame []byte) ([]byte, error) {
	return s.r.add(frame)
}

// SMPHeader is an SMP header as the external tests see it.
type SMPHeader struct {
	Op    uint8
	Group uint16
	Seq   uint8
	ID    uint8
}

// SMPParse splits a packet into header and CBOR body.
func SMPParse(pkt []byte) (SMPHeader, []byte, error) {
	h, body, err := smpParsePacket(pkt)
	return SMPHeader{Op: h.op, Group: h.group, Seq: h.seq, ID: h.id}, body, err
}

// SMPPacket builds a packet with a CBOR body.
func SMPPacket(h SMPHeader, body any) ([]byte, error) {
	return smpPacket(h.Op, h.Group, h.ID, h.Seq, body)
}
