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

	minPacketLen := minPacketLen[ret.FunctionCode]
	if minPacketLen == 0 {
		return PDU{}, fmt.Errorf("unsupported Function code: %v",
			ret.FunctionCode)
	}

	if len(packet) < minPacketLen {
		return PDU{}, fmt.Errorf("not enough data for function code %v, expected %v, got %v", ret.FunctionCode, minPacketLen, len(packet))
	}

	ret.Data = packet[2 : len(packet)-2]

	return ret, nil
}
