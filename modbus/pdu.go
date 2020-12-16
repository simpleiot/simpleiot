package modbus

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// PDU for Modbus packets
type PDU struct {
	FunctionCode FunctionCode
	Data         []byte
}

func (p PDU) String() string {
	return fmt.Sprintf("PDU: %v: %v", p.FunctionCode,
		HexDump(p.Data))
}

//RegChange is a type that describes a modbus register change
// the address, old value and new value of the register are provided.
// This allows application software to take action when things change.
type RegChange struct {
	Address uint16
	Old     uint16
	New     uint16
}

// ProcessRequest a modbus request. Registers are read and written
// through the server interface argument.
// This function returns any register changes, the modbus respose,
// and any errors
func (p *PDU) ProcessRequest(regs *Regs) ([]RegChange, PDU, error) {
	changes := []RegChange{}
	resp := PDU{}
	resp.FunctionCode = p.FunctionCode

	switch p.FunctionCode {
	case FuncCodeReadCoils, FuncCodeReadDiscreteInputs:
		address := binary.BigEndian.Uint16(p.Data[:2])
		count := binary.BigEndian.Uint16(p.Data[2:4])
		// FIXME, do something with count to handle a range of reads
		_ = count
		v, err := regs.ReadReg(address / 16)
		if err != nil {
			return []RegChange{}, PDU{}, errors.New(
				"Did not find modbus reg")
		}
		bitPos := address % 16
		bitV := (v >> uint(bitPos)) & 0x1
		resp.Data = []byte{1, byte(bitV)}
	case FuncCodeReadHoldingRegisters, FuncCodeReadInputRegisters:
		address := binary.BigEndian.Uint16(p.Data[:2])
		count := binary.BigEndian.Uint16(p.Data[2:4])

		resp.Data = make([]byte, 1+2*count)
		resp.Data[0] = uint8(count * 2)
		for i := 0; i < int(count); i++ {
			v, err := regs.ReadReg(address + uint16(i))
			if err != nil {
				return []RegChange{}, PDU{}, errors.New(
					"Did not find modbus reg")
			}

			binary.BigEndian.PutUint16(resp.Data[1+i*2:], v)

		}

	default:
		return []RegChange{}, PDU{},
			fmt.Errorf("unsupported function code: %v", p.FunctionCode)

	}

	return changes, resp, nil
}

// RespReadBits reads coils and discrete inputs from a
// response PDU.
func (p *PDU) RespReadBits() ([]bool, error) {
	if len(p.Data) < 2 {
		return []bool{}, errors.New("not enough data")
	}
	switch p.FunctionCode {
	case FuncCodeReadCoils, FuncCodeReadDiscreteInputs:
		// ok
	default:
		return []bool{}, errors.New("invalid function code to read bits")
	}

	count := p.Data[0]
	ret := make([]bool, count)
	byteIndex := 0
	bitIndex := uint(0)

	for i := byte(0); i < count; i++ {
		ret[i] = ((p.Data[byteIndex+1] >> bitIndex) & 0x1) == 0x1
		bitIndex++
		if bitIndex >= 8 {
			byteIndex++
			bitIndex = 0
		}
	}

	return ret, nil
}

// RespReadRegs reads register values from a
// response PDU.
func (p *PDU) RespReadRegs() ([]uint16, error) {
	if len(p.Data) < 2 {
		return []uint16{}, errors.New("not enough data")
	}
	switch p.FunctionCode {
	case FuncCodeReadHoldingRegisters, FuncCodeReadInputRegisters:
		// ok
	default:
		return []uint16{}, errors.New("invalid function code to read regs")
	}

	count := p.Data[0] / 2

	if len(p.Data) < 1+int(count)*2 {
		return []uint16{}, errors.New("RespReadRegs not enough data")
	}

	ret := make([]uint16, count)

	for i := 0; i < int(count); i++ {
		ret[i] = binary.BigEndian.Uint16(p.Data[1+i*2 : 1+i*2+2])
	}

	return ret, nil
}

// Add address units below are the packet address, typically drop
// first digit from register and subtract 1

// ReadDiscreteInputs creates PDU to read descrete inputs
func ReadDiscreteInputs(address uint16, count uint16) PDU {
	return PDU{
		FunctionCode: FuncCodeReadDiscreteInputs,
		Data:         PutUint16Array(address, count),
	}
}

// ReadCoils creates PDU to read coils
func ReadCoils(address uint16, count uint16) PDU {
	return PDU{
		FunctionCode: FuncCodeReadCoils,
		Data:         PutUint16Array(address, count),
	}
}

// ReadHoldingRegs creates a PDU to read a holding regs
func ReadHoldingRegs(address uint16, count uint16) PDU {
	return PDU{
		FunctionCode: FuncCodeReadHoldingRegisters,
		Data:         PutUint16Array(address, count),
	}
}

// ReadInputRegs creates a PDU to read input regs
func ReadInputRegs(address uint16, count uint16) PDU {
	return PDU{
		FunctionCode: FuncCodeReadInputRegisters,
		Data:         PutUint16Array(address, count),
	}
}
