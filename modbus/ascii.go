package modbus

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

const (
	asciiStart   = ':'
	asciiEnd     = "\r\n"
	asciiMinSize = 9
	asciiMaxSize = 513
)

// Modbus is a type that implements modbus ascii communication.
// Currently, only "sniffing" a network is implemented
type Modbus struct {
	io      io.ReadWriter
	bufRead *bufio.Reader
}

// NewModbus creates a new Modbus
func NewModbus(port io.ReadWriter) *Modbus {
	return &Modbus{
		io:      port,
		bufRead: bufio.NewReader(port),
	}
}

// Read returns an ASCII modbus packet. Blocks until
// a full packet is received or error
func (m *Modbus) Read() ([]byte, error) {
	return m.bufRead.ReadBytes(0xA)
}

// PDU is a modbus protocol data unit
type PDU struct {
	Address      byte
	FunctionCode FunctionCode
	Data         []byte
	LRC          byte
	End          []byte // should be "\r\n"
}

// CheckLRC verifies the LRC is valid
func (pdu *PDU) CheckLRC() bool {
	var sum byte
	sum += pdu.Address
	sum += byte(pdu.FunctionCode)
	for _, b := range pdu.Data {
		sum += b
	}

	return byte(-int8(sum)) == pdu.LRC
}

// DecodeFunctionData extracts the function data from the PDU
func (pdu *PDU) DecodeFunctionData() (ret interface{}, err error) {
	switch pdu.FunctionCode {
	case FuncCodeWriteMultipleRegisters:
		if len(pdu.Data) < 5 {
			err = errors.New("not enough data for Write Mult Regs")
			return
		}
		r := FuncWriteMultipleRegisterRequest{}
		r.FunctionCode = pdu.FunctionCode
		r.StartingAddress = uint16(pdu.Data[0])<<8 | uint16(pdu.Data[1])
		r.RegCount = uint16(pdu.Data[2])<<8 | uint16(pdu.Data[3])
		r.ByteCount = pdu.Data[4]
		if r.RegCount*2 != uint16(r.ByteCount) {
			err = errors.New("Byte count does not match reg count")
			return
		}
		regData := pdu.Data[5:]
		if len(regData) != int(r.ByteCount) {
			err = errors.New("not enough reg data")
			return
		}
		for i := 0; i < int(r.RegCount); i++ {
			v := uint16(regData[i*2])<<8 | uint16(regData[i*2+1])
			r.RegValues = append(r.RegValues, v)
		}
		ret = r
	default:
		err = fmt.Errorf("Unhandled function code %v", pdu.FunctionCode)
	}

	return
}

// FuncReadHoldingRegistersRequest represents the request to read holding reg
type FuncReadHoldingRegistersRequest struct {
	FunctionCode    FunctionCode
	StartingAddress uint16
	RegCount        uint16
}

// FuncReadHoldingRegisterResponse response to read holding reg
type FuncReadHoldingRegisterResponse struct {
	FunctionCode FunctionCode
	RegCount     byte
	RegValues    []uint16
}

// FuncWriteMultipleRegisterRequest represents the request to write multiple regs
type FuncWriteMultipleRegisterRequest struct {
	FunctionCode    FunctionCode
	StartingAddress uint16
	RegCount        uint16
	ByteCount       byte
	RegValues       []uint16
}

// DecodeASCIIByte converts type ascii hex bytes to a binary
// byte
func DecodeASCIIByte(data []byte) (byte, []byte, error) {
	if len(data) < 2 {
		return 0, []byte{}, errors.New("Not enough data to decode")
	}

	ret := make([]byte, 1)
	_, err := hex.Decode(ret, data[:2])
	if err != nil {
		return 0, []byte{}, err
	}

	return ret[0], data[2:], nil
}

// DecodeASCIIByteEnd converts type ascii hex bytes to a binary
// byte. This function takes from the end of the slice
func DecodeASCIIByteEnd(data []byte) (byte, []byte, error) {
	if len(data) < 2 {
		return 0, []byte{}, errors.New("Not enough data to decode")
	}

	ret := make([]byte, 1)
	_, err := hex.Decode(ret, data[len(data)-2:])
	if err != nil {
		return 0, []byte{}, err
	}

	return ret[0], data[:len(data)-2], nil
}

// DecodeASCIIPDU decodes a ASCII modbus packet
func DecodeASCIIPDU(data []byte) (ret PDU, err error) {
	if len(data) < asciiMinSize {
		err = errors.New("not enough data to decode")
		return
	}

	if data[0] != asciiStart {
		return PDU{}, errors.New("invalid start char")
	}

	// chop start
	data = data[1:]

	cnt := len(data)
	ret.End = make([]byte, 2)
	copy(ret.End, data[cnt-2:])

	if string(ret.End) != asciiEnd {
		err = fmt.Errorf("ending is not correct: %v", ret.End)
		return
	}

	// chop end
	data = data[:cnt-2]

	// pop address and function code off the front end of the data
	ret.Address, data, err = DecodeASCIIByte(data)
	if err != nil {
		return
	}

	var fc byte
	fc, data, err = DecodeASCIIByte(data)
	ret.FunctionCode = FunctionCode(fc)
	if err != nil {
		return
	}

	// pop LRC off the end of the data
	ret.LRC, data, err = DecodeASCIIByteEnd(data)
	if err != nil {
		return
	}

	// what we are left with is the data payload
	ret.Data = make([]byte, hex.DecodedLen(len(data)))
	_, err = hex.Decode(ret.Data, data)
	if err != nil {
		return
	}

	if !ret.CheckLRC() {
		err = errors.New("LRC check failed")
	}

	return
}
