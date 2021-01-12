package modbus

import (
	"encoding/binary"
	"fmt"
)

// RtuADU defines an ADU for RTU packets
type RtuADU struct {
	PDU
	Address byte
	CRC     uint16
}

// RtuEncode encodes a RTU packet
func RtuEncode(id byte, pdu PDU) ([]byte, error) {
	ret := make([]byte, len(pdu.Data)+2+2)
	ret[0] = id
	ret[1] = byte(pdu.FunctionCode)
	copy(ret[2:], pdu.Data)
	crc := RtuCrc(ret[:len(ret)-2])
	binary.BigEndian.PutUint16(ret[len(ret)-2:], crc)
	return ret, nil
}

// RtuDecode decodes a RTU packet
func RtuDecode(packet []byte) (PDU, error) {
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
