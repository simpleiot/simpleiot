package modbus

import (
	"encoding/binary"
	"fmt"
	"io"
)

// TCPADU defines an ADU for TCP packets
type TCPADU struct {
	PDU
	Address byte
}

// TCP defines an TCP connection
type TCP struct {
	port io.ReadWriter
	txID uint16
}

// NewTCP creates a new TCP transport
func NewTCP(port io.ReadWriter) *TCP {
	return &TCP{
		port: port,
	}
}

func (r *TCP) Read(p []byte) (int, error) {
	return r.port.Read(p)
}

func (r *TCP) Write(p []byte) (int, error) {
	return r.port.Write(p)
}

// Encode encodes a RTU packet
func (r *TCP) Encode(id byte, pdu PDU) ([]byte, error) {
	// increment transaction ID
	r.txID++
	// bytes 0,1 transaction ID
	ret := make([]byte, len(pdu.Data)+8)
	binary.BigEndian.PutUint16(ret[0:], r.txID)

	// bytes 2,3 protocol identifier

	// bytes 4,5 length
	binary.BigEndian.PutUint16(ret[4:], uint16(len(pdu.Data)+2))

	// byte 6 unit identifier
	ret[6] = id

	// byte 7 function code
	ret[7] = byte(pdu.FunctionCode)

	// byte 8: data
	copy(ret[8:], pdu.Data)
	return ret, nil
}

// Decode decodes a RTU packet
func (r *TCP) Decode(packet []byte) (PDU, error) {
	if len(packet) < 9 {
		return PDU{}, fmt.Errorf("Not enough data for TCP packet: %v", len(packet))
	}

	// FIXME check txID

	ret := PDU{}

	ret.FunctionCode = FunctionCode(packet[7])

	ret.Data = packet[8:]

	return ret, nil
}
