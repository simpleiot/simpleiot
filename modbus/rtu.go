package modbus

import (
	"encoding/binary"
	"fmt"
	"io"
)

// RtuADU defines an ADU for RTU packets
type RtuADU struct {
	PDU
	Address byte
	CRC     uint16
}

// RTU defines an RTU connection
type RTU struct {
	port io.ReadWriteCloser
}

// NewRTU creates a new RTU transport
func NewRTU(port io.ReadWriteCloser) *RTU {
	return &RTU{
		port: port,
	}
}

func (r *RTU) Read(p []byte) (int, error) {
	return r.port.Read(p)
}

func (r *RTU) Write(p []byte) (int, error) {
	return r.port.Write(p)
}

// Close closes the serial port
func (r *RTU) Close() error {
	return r.port.Close()
}

// Encode encodes a RTU packet
func (r *RTU) Encode(id byte, pdu PDU) ([]byte, error) {
	ret := make([]byte, len(pdu.Data)+2+2)
	ret[0] = id
	ret[1] = byte(pdu.FunctionCode)
	copy(ret[2:], pdu.Data)
	crc := RtuCrc(ret[:len(ret)-2])
	binary.BigEndian.PutUint16(ret[len(ret)-2:], crc)
	return ret, nil
}

// Decode decodes a RTU packet
func (r *RTU) Decode(packet []byte) (byte, PDU, error) {
	err := CheckRtuCrc(packet)
	if err != nil {
		return 0, PDU{}, err
	}

	ret := PDU{}

	ret.FunctionCode = FunctionCode(packet[1])

	if len(packet) < 4 {
		return 0, PDU{}, fmt.Errorf("short packet, got %d bytes", len(packet))
	}

	id := packet[0]

	ret.Data = packet[2 : len(packet)-2]

	return id, ret, nil
}

// Type returns TransportType
func (r *RTU) Type() TransportType {
	return TransportTypeRTU
}
