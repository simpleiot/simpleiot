package modbus

import "encoding/binary"

// FunctionCode represents a modbus function code
type FunctionCode byte

// Defined valid function codes
const (
	// Bit access
	FuncCodeReadDiscreteInputs FunctionCode = 2
	FuncCodeReadCoils                       = 1
	FuncCodeWriteSingleCoil                 = 5
	FuncCodeWriteMultipleCoils              = 15

	// 16-bit access
	FuncCodeReadInputRegisters         = 4
	FuncCodeReadHoldingRegisters       = 3
	FuncCodeWriteSingleRegister        = 6
	FuncCodeWriteMultipleRegisters     = 16
	FuncCodeReadWriteMultipleRegisters = 23
	FuncCodeMaskWriteRegister          = 22
	FuncCodeReadFIFOQueue              = 24
)

var minPacketLen = map[FunctionCode]int{
	FuncCodeReadCoils: 8,
}

// dataBlock creates a sequence of uint16 data.
func dataBlock(value ...uint16) []byte {
	data := make([]byte, 2*len(value))
	for i, v := range value {
		binary.BigEndian.PutUint16(data[i*2:], v)
	}
	return data
}
