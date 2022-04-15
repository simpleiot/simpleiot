package modbus

import (
	"encoding/binary"
	"math"
)

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

// RegsToInt16 converts modbus regs to int16 values
func RegsToInt16(in []uint16) []int16 {
	ret := make([]int16, len(in))
	for i := range in {
		ret[i] = int16(in[i])
	}

	return ret
}

// RegsToUint32 converts modbus regs to uint32 values
func RegsToUint32(in []uint16) []uint32 {
	count := len(in) / 2
	ret := make([]uint32, count)
	for i := range ret {
		buf := make([]byte, 4)
		binary.BigEndian.PutUint16(buf[0:], in[i*2])
		binary.BigEndian.PutUint16(buf[2:], in[i*2+1])
		ret[i] = binary.BigEndian.Uint32(buf)
	}

	return ret
}

// Uint32ToRegs converts uint32 values to modbus regs
func Uint32ToRegs(in []uint32) []uint16 {
	ret := make([]uint16, len(in)*2)
	for i, v := range in {
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, v)
		ret[i*2] = binary.BigEndian.Uint16(buf[0:])
		ret[i*2+1] = binary.BigEndian.Uint16(buf[2:])
	}

	return ret
}

// RegsToInt32 converts modbus regs to int32 values
func RegsToInt32(in []uint16) []int32 {
	count := len(in) / 2
	ret := make([]int32, count)
	for i := range ret {
		buf := make([]byte, 4)
		binary.BigEndian.PutUint16(buf[0:], in[i*2])
		binary.BigEndian.PutUint16(buf[2:], in[i*2+1])
		ret[i] = int32(binary.BigEndian.Uint32(buf))
	}

	return ret
}

// Int32ToRegs converts int32 values to modbus regs
func Int32ToRegs(in []int32) []uint16 {
	ret := make([]uint16, len(in)*2)
	for i, v := range in {
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, uint32(v))
		ret[i*2] = binary.BigEndian.Uint16(buf[0:])
		ret[i*2+1] = binary.BigEndian.Uint16(buf[2:])
	}

	return ret
}

// RegsToFloat32SwapWords converts modbus regs to float32 values
func RegsToFloat32SwapWords(in []uint16) []float32 {
	count := len(in) / 2
	ret := make([]float32, count)
	for i := range ret {
		buf := make([]byte, 4)
		binary.BigEndian.PutUint16(buf[2:], in[i*2])
		binary.BigEndian.PutUint16(buf[0:], in[i*2+1])
		ret[i] = math.Float32frombits(binary.BigEndian.Uint32(buf))
	}

	return ret
}

// RegsToFloat32 converts modbus regs to float32 values
func RegsToFloat32(in []uint16) []float32 {
	count := len(in) / 2
	ret := make([]float32, count)
	for i := range ret {
		buf := make([]byte, 4)
		binary.BigEndian.PutUint16(buf[0:], in[i*2])
		binary.BigEndian.PutUint16(buf[2:], in[i*2+1])
		ret[i] = math.Float32frombits(binary.BigEndian.Uint32(buf))
	}

	return ret
}

// Float32ToRegs converts float32 values to modbus regs
func Float32ToRegs(in []float32) []uint16 {
	ret := make([]uint16, len(in)*2)
	for i, v := range in {
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, math.Float32bits(v))
		ret[i*2] = binary.BigEndian.Uint16(buf[0:])
		ret[i*2+1] = binary.BigEndian.Uint16(buf[2:])
	}

	return ret
}
