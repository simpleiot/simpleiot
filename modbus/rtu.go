package modbus

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
	ret[len(ret)-2] = byte(crc & 0xff)
	ret[len(ret)-1] = byte(crc >> 8)
	return ret, nil
}
