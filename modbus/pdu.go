package modbus

import (
	"encoding/binary"
	"errors"
	"fmt"
	"slices"

	"github.com/simpleiot/simpleiot/test"
)

// Protocol limits on the number of items one request may address
// (Modbus Application Protocol V1.1b3, section 6).
const (
	MaxReadBits  = 2000
	MaxReadRegs  = 125
	MaxWriteBits = 1968
	MaxWriteRegs = 123
)

// PDU for Modbus packets
type PDU struct {
	FunctionCode FunctionCode
	Data         []byte
}

func (p PDU) String() string {
	return fmt.Sprintf("PDU: %v: %v", p.FunctionCode,
		test.HexDump(p.Data))
}

// handleError translates an error into a PDU, if possible.
func (p *PDU) handleError(err error) (bool, PDU, error) {
	if err, ok := err.(ExceptionCode); ok {
		resp := PDU{}
		resp.FunctionCode = p.FunctionCode | 0x80
		resp.Data = []byte{byte(err)}
		return false, resp, nil
	}
	// TODO: Wrap the underlying error?
	return p.handleError(ExcServerDeviceFailure)
}

// ProcessRequest a modbus request. Registers are read and written
// through the server interface argument.
// This function returns any register changes, the modbus response,
// and any errors
func (p *PDU) ProcessRequest(regs RegProvider) (bool, PDU, error) {
	regsChanged := false
	resp := PDU{}
	resp.FunctionCode = p.FunctionCode

	minPacketLen := minRequestLen[p.FunctionCode]

	if len(p.Data) < minPacketLen-1 {
		return false, PDU{}, fmt.Errorf("not enough data for function code %v, expected %v, got %v", p.FunctionCode, minPacketLen, len(p.Data))
	}

	switch p.FunctionCode {
	case FuncCodeReadCoils, FuncCodeReadDiscreteInputs:
		address := binary.BigEndian.Uint16(p.Data[:2])
		count := binary.BigEndian.Uint16(p.Data[2:4])
		if count < 1 || count > MaxReadBits {
			return p.handleError(ExcIllegalValue)
		}
		bytes := byte((count + 7) / 8)
		resp.Data = make([]byte, 1+bytes)
		resp.Data[0] = bytes
		var read = regs.ReadCoil
		if p.FunctionCode == FuncCodeReadDiscreteInputs {
			read = regs.ReadDiscreteInput
		}
		for i := 0; i < int(count); i++ {
			v, err := read(int(address) + i)
			if err != nil {
				return p.handleError(err)
			}
			if v {
				resp.Data[1+i/8] |= 1 << (i % 8)
			}
		}
	case FuncCodeReadHoldingRegisters, FuncCodeReadInputRegisters:
		address := binary.BigEndian.Uint16(p.Data[:2])
		count := binary.BigEndian.Uint16(p.Data[2:4])
		if count < 1 || count > MaxReadRegs {
			return p.handleError(ExcIllegalValue)
		}

		resp.Data = make([]byte, 1+2*count)
		resp.Data[0] = uint8(count * 2)
		var read = regs.ReadReg
		if p.FunctionCode == FuncCodeReadInputRegisters {
			read = regs.ReadInputReg
		}
		for i := 0; i < int(count); i++ {
			v, err := read(int(address) + i)
			if err != nil {
				return p.handleError(err)
			}

			binary.BigEndian.PutUint16(resp.Data[1+i*2:], v)

		}

	case FuncCodeWriteSingleCoil:
		address := binary.BigEndian.Uint16(p.Data[:2])
		v := binary.BigEndian.Uint16(p.Data[2:4])

		vBool := false
		switch v {
		case WriteCoilValueOff:
		case WriteCoilValueOn:
			vBool = true
		default:
			return p.handleError(ExcIllegalValue)
		}

		err := regs.WriteCoil(int(address), vBool)
		if err != nil {
			return p.handleError(err)
		}

		regsChanged = true
		resp.Data = p.Data

	case FuncCodeWriteMultipleCoils:
		address := binary.BigEndian.Uint16(p.Data[:2])
		quantity := binary.BigEndian.Uint16(p.Data[2:4])
		if quantity < 1 || quantity > MaxWriteBits {
			return p.handleError(ExcIllegalValue)
		}
		if len(p.Data) != 5+((int(quantity)+7)/8) {
			return p.handleError(ExcIllegalValue)
		}
		for i := 0; i < int(quantity); i++ {
			value := (p.Data[5+i/8]>>(i%8))&1 == 1
			if err := regs.WriteCoil(int(address)+i, value); err != nil {
				return p.handleError(err)
			}
		}
		resp.Data = make([]byte, 4)
		binary.BigEndian.PutUint16(resp.Data[:2], address)
		binary.BigEndian.PutUint16(resp.Data[2:4], quantity)
		regsChanged = true

	case FuncCodeWriteSingleRegister:
		address := binary.BigEndian.Uint16(p.Data[:2])
		v := binary.BigEndian.Uint16(p.Data[2:4])

		err := regs.WriteReg(int(address), v)
		if err != nil {
			return p.handleError(err)
		}

		resp = *p
		regsChanged = true

	case FuncCodeWriteMultipleRegisters:
		address := binary.BigEndian.Uint16(p.Data[:2])
		quantity := binary.BigEndian.Uint16(p.Data[2:4])
		if quantity < 1 || quantity > MaxWriteRegs {
			return p.handleError(ExcIllegalValue)
		}
		if len(p.Data) != 5+(int(quantity)*2) {
			return p.handleError(ExcIllegalValue)
		}
		for i := 0; i < int(quantity); i++ {
			value := binary.BigEndian.Uint16(p.Data[5+i*2 : 5+i*2+2])
			if err := regs.WriteReg(int(address)+i, value); err != nil {
				return p.handleError(err)
			}
		}
		resp.Data = make([]byte, 4)
		binary.BigEndian.PutUint16(resp.Data[:2], address)
		binary.BigEndian.PutUint16(resp.Data[2:4], quantity)
		regsChanged = true

	default:
		return p.handleError(ExcIllegalFunction)
	}

	return regsChanged, resp, nil
}

// respException returns the exception a response carries, if any.
func (p *PDU) respException() error {
	if p.FunctionCode&0x80 == 0 {
		return nil
	}
	if len(p.Data) < 1 {
		return errors.New("exception response without a code")
	}
	return ExceptionCode(p.Data[0])
}

// RespReadBits reads coils and discrete inputs from a response PDU. count is
// the number of bits the request asked for; the response must carry exactly
// that many, so a server cannot make the client read past its buffer.
func (p *PDU) RespReadBits(count uint16) ([]bool, error) {
	if err := p.respException(); err != nil {
		return []bool{}, err
	}
	switch p.FunctionCode {
	case FuncCodeReadCoils, FuncCodeReadDiscreteInputs:
		// ok
	default:
		return []bool{}, errors.New("invalid function code to read bits")
	}
	if count < 1 || count > MaxReadBits {
		return []bool{}, fmt.Errorf("bit count %v out of range", count)
	}
	if len(p.Data) < 1 {
		return []bool{}, errors.New("not enough data")
	}

	byteCount := int(count+7) / 8
	if int(p.Data[0]) != byteCount {
		return []bool{}, fmt.Errorf("expected byte count %v, got %v",
			byteCount, p.Data[0])
	}
	if len(p.Data) < 1+byteCount {
		return []bool{}, fmt.Errorf("expected %v data bytes, got %v",
			byteCount, len(p.Data)-1)
	}

	ret := make([]bool, count)
	for i := range ret {
		ret[i] = (p.Data[1+i/8]>>(i%8))&0x1 == 0x1
	}

	return ret, nil
}

// RespReadRegs reads register values from a response PDU. count is the
// number of registers the request asked for; the response must carry
// exactly that many.
func (p *PDU) RespReadRegs(count uint16) ([]uint16, error) {
	if err := p.respException(); err != nil {
		return []uint16{}, err
	}
	switch p.FunctionCode {
	case FuncCodeReadHoldingRegisters, FuncCodeReadInputRegisters:
		// ok
	default:
		return []uint16{}, errors.New("invalid function code to read regs")
	}
	if count < 1 || count > MaxReadRegs {
		return []uint16{}, fmt.Errorf("register count %v out of range", count)
	}
	if len(p.Data) < 1 {
		return []uint16{}, errors.New("not enough data")
	}

	byteCount := int(count) * 2
	if int(p.Data[0]) != byteCount {
		return []uint16{}, fmt.Errorf("expected byte count %v, got %v",
			byteCount, p.Data[0])
	}
	if len(p.Data) < 1+byteCount {
		return []uint16{}, fmt.Errorf("expected %v data bytes, got %v",
			byteCount, len(p.Data)-1)
	}

	ret := make([]uint16, count)
	for i := range ret {
		ret[i] = binary.BigEndian.Uint16(p.Data[1+i*2 : 1+i*2+2])
	}

	return ret, nil
}

// Add address units below are the packet address, typically drop
// first digit from register and subtract 1

// ReadDiscreteInputs creates PDU to read discrete inputs
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

// WriteSingleCoil creates PDU to read coils
func WriteSingleCoil(address uint16, v bool) PDU {
	value := WriteCoilValueOff
	if v {
		value = WriteCoilValueOn
	}

	return PDU{
		FunctionCode: FuncCodeWriteSingleCoil,
		Data:         PutUint16Array(address, value),
	}
}

// WriteSingleReg creates PDU to write a single holding reg
func WriteSingleReg(address, value uint16) PDU {
	return PDU{
		FunctionCode: FuncCodeWriteSingleRegister,
		Data:         PutUint16Array(address, value),
	}
}

// WriteMultipleRegs creates PDU to write multiple holding regs
func WriteMultipleRegs(address uint16, quantity uint16, values []uint16) PDU {
	return PDU{
		FunctionCode: FuncCodeWriteMultipleRegisters,
		Data:         putUint16ArrayWithByteCount(address, quantity, values),
	}
}

func putUint16ArrayWithByteCount(address uint16, quantity uint16, values []uint16) []byte {
	addressAndQuantity := PutUint16Array(address, quantity)
	byteCount := []byte{byte(uint8(quantity * 2))}
	data := PutUint16Array(values...)
	return slices.Concat(addressAndQuantity, byteCount, data)
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
