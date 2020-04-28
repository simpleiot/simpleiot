package modbus

import "encoding/binary"

// PutUint16Array creates a sequence of uint16 data.
func PutUint16Array(value ...uint16) []byte {
	data := make([]byte, 2*len(value))
	for i, v := range value {
		binary.BigEndian.PutUint16(data[i*2:], v)
	}
	return data
}

// Uint16Array unpacks 16 bit data values from a buffer
// (in big endian format)
func Uint16Array(data []byte) []uint16 {
	ret := make([]uint16, len(data)/2)
	for i := range ret {
		ret[i] = binary.BigEndian.Uint16(data[i*2 : i*2+2])
	}
	return ret
}
