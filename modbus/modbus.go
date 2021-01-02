package modbus

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

// define valid values for write coil
const (
	WriteCoilValueOn  uint16 = 0xff00
	WriteCoilValueOff uint16 = 0
)

var minPacketLen = map[FunctionCode]int{
	FuncCodeReadCoils:            6,
	FuncCodeReadDiscreteInputs:   6,
	FuncCodeReadHoldingRegisters: 7,
	FuncCodeReadInputRegisters:   7,
	FuncCodeWriteSingleCoil:      7,
	FuncCodeWriteSingleRegister:  7,
}
