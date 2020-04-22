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
		// FIXME, do something with count
		_ = count
		v, err := regs.ReadReg(address / 16)
		if err != nil {
			return []RegChange{}, PDU{}, errors.New(
				"Did not find modbus reg")
		}
		bitPos := address % 16
		bitV := (v >> bitPos) & 0x1
		resp.Data = []byte{1, byte(bitV)}
	default:
		return []RegChange{}, PDU{},
			fmt.Errorf("unsupported function code: %v", p.FunctionCode)

	}

	return changes, resp, nil
}

// RespReadBits reads coils and discrete inputs from a
// response PDU.
func (p *PDU) RespReadBits() ([]bool, error) {
	switch p.FunctionCode {
	case FuncCodeReadCoils, FuncCodeReadDiscreteInputs:
		// ok
	default:
		return []bool{}, errors.New("invalid function code to read bits")
	}

	count := p.Data[0]
	ret := make([]bool, count)
	byteIndex := 0
	bitIndex := 0

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

// Add address units below are the packet address, typically drop
// first digit from register and subtract 1

// ReadCoils creates PDU to read coils
func ReadCoils(address uint16, count uint16) PDU {
	return PDU{
		FunctionCode: FuncCodeReadCoils,
		Data:         dataBlock(address, count),
	}
}
