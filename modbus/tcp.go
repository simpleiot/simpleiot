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
	CRC     uint16
}

// TCP defines an TCP connection
type TCP struct {
	port io.ReadWriter
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
	ret := make([]byte, len(pdu.Data)+2+2)
	ret[0] = id
	ret[1] = byte(pdu.FunctionCode)
	copy(ret[2:], pdu.Data)
	crc := RtuCrc(ret[:len(ret)-2])
	binary.BigEndian.PutUint16(ret[len(ret)-2:], crc)
	return ret, nil
}

// Decode decodes a RTU packet
func (r *TCP) Decode(packet []byte) (PDU, error) {
	err := CheckRtuCrc(packet)
	if err != nil {
		return PDU{}, err
	}

	ret := PDU{}

	ret.FunctionCode = FunctionCode(packet[1])

	if len(packet) < 4 {
		return PDU{}, fmt.Errorf("short packet, got %d bytes", len(packet))
	}

	ret.Data = packet[2 : len(packet)-2]

	return ret, nil
}
