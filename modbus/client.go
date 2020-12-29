package modbus

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// Client defines a Modbus client (master)
type Client struct {
	port  io.ReadWriter
	debug int
}

// NewClient is used to create a new modbus client
// port must return an entire packet for each Read().
// github.com/simpleiot/simpleiot/respreader is a good
// way to do this.
func NewClient(port io.ReadWriter, debug int) *Client {
	return &Client{
		port:  port,
		debug: debug,
	}
}

// SetDebugLevel allows you to change debug level on the fly
func (c *Client) SetDebugLevel(debug int) {
	c.debug = debug
}

// ReadCoils is used to read modbus coils
func (c *Client) ReadCoils(id byte, coil, count uint16) ([]bool, error) {
	ret := []bool{}
	req := ReadCoils(coil, count)
	if c.debug >= 1 {
		fmt.Println("Modbus client Readcoils req: ", req)
	}
	packet, err := RtuEncode(id, req)
	if err != nil {
		return ret, err
	}

	if c.debug >= 9 {
		fmt.Println("Modbus client ReadCoils tx: ", HexDump(packet))
	}

	_, err = c.port.Write(packet)
	if err != nil {
		return ret, err
	}

	// FIXME, what is max modbus packet size?
	buf := make([]byte, 200)
	cnt, err := c.port.Read(buf)
	if err != nil {
		return ret, err
	}

	buf = buf[:cnt]

	if c.debug >= 9 {
		fmt.Println("Modbus client ReadCoils rx: ", HexDump(buf))
	}

	resp, err := RtuDecode(buf)
	if err != nil {
		return ret, err
	}

	if c.debug >= 1 {
		fmt.Println("Modbus client Readcoils resp: ", resp)
	}

	return resp.RespReadBits()
}

// WriteSingleCoil is used to read modbus coils
func (c *Client) WriteSingleCoil(id byte, coil uint16, v bool) error {
	req := WriteSingleCoil(coil, v)
	if c.debug >= 1 {
		fmt.Println("Modbus client WriteSingleCoil req: ", req)
	}
	packet, err := RtuEncode(id, req)
	if err != nil {
		return err
	}

	if c.debug >= 9 {
		fmt.Println("Modbus client WriteSingleCoil tx: ", HexDump(packet))
	}

	_, err = c.port.Write(packet)
	if err != nil {
		return err
	}

	// FIXME, what is max modbus packet size?
	buf := make([]byte, 200)
	cnt, err := c.port.Read(buf)
	if err != nil {
		return err
	}

	buf = buf[:cnt]

	if c.debug >= 9 {
		fmt.Println("Modbus client WriteSingleCoil rx: ", HexDump(buf))
	}

	resp, err := RtuDecode(buf)
	if err != nil {
		return err
	}

	// FIXME, check return code matches what we sent
	if resp.FunctionCode != FuncCodeWriteSingleCoil {
		return errors.New("resp contains wrong function code")
	}

	if len(resp.Data) < 2 {
		return errors.New("not enough data in resp")
	}

	ret := binary.BigEndian.Uint16(resp.Data)

	exp := WriteCoilValueOff
	if v {
		exp = WriteCoilValueOn
	}

	if ret != exp {
		return errors.New("Write coil did not return expected value")
	}

	if c.debug >= 1 {
		fmt.Println("Modbus client WriteSingleCoil resp: ", resp)
	}

	return nil
}

// ReadDiscreteInputs is used to read modbus discrete inputs
func (c *Client) ReadDiscreteInputs(id byte, input, count uint16) ([]bool, error) {
	ret := []bool{}
	req := ReadDiscreteInputs(input, count)
	if c.debug >= 1 {
		fmt.Println("Modbus client ReadDiscreteInputs req: ", req)
	}
	packet, err := RtuEncode(id, req)
	if err != nil {
		return ret, err
	}

	if c.debug >= 9 {
		fmt.Println("Modbus client ReadDiscreteInputs tx: ", HexDump(packet))
	}

	_, err = c.port.Write(packet)
	if err != nil {
		return ret, err
	}

	// FIXME, what is max modbus packet size?
	buf := make([]byte, 200)
	cnt, err := c.port.Read(buf)
	if err != nil {
		return ret, err
	}

	buf = buf[:cnt]

	if c.debug >= 9 {
		fmt.Println("Modbus client ReadDiscreteInputs rx: ", HexDump(buf))
	}

	resp, err := RtuDecode(buf)
	if err != nil {
		return ret, err
	}

	if c.debug >= 1 {
		fmt.Println("Modbus client ReadDiscreteInputs resp: ", resp)
	}

	return resp.RespReadBits()
}

// ReadHoldingRegs is used to read modbus coils
func (c *Client) ReadHoldingRegs(id byte, reg, count uint16) ([]uint16, error) {
	ret := []uint16{}
	req := ReadHoldingRegs(reg, count)
	if c.debug >= 1 {
		fmt.Println("Modbus client ReadHoldingRegs req: ", req)
	}
	packet, err := RtuEncode(id, req)
	if err != nil {
		return ret, err
	}

	if c.debug >= 9 {
		fmt.Println("Modbus client ReadHoldingRegs tx: ", HexDump(packet))
	}

	_, err = c.port.Write(packet)
	if err != nil {
		return ret, err
	}

	// FIXME, what is max modbus packet size?
	buf := make([]byte, 200)
	cnt, err := c.port.Read(buf)
	if err != nil {
		return ret, err
	}

	buf = buf[:cnt]

	if c.debug >= 9 {
		fmt.Println("Modbus client ReadHoldingRegs rx: ", HexDump(buf))
	}

	resp, err := RtuDecode(buf)
	if err != nil {
		return ret, err
	}

	if c.debug >= 1 {
		fmt.Println("Modbus client ReadHoldingRegs resp: ", resp)
	}

	return resp.RespReadRegs()
}

// ReadInputRegs is used to read modbus coils
func (c *Client) ReadInputRegs(id byte, reg, count uint16) ([]uint16, error) {
	ret := []uint16{}
	req := ReadInputRegs(reg, count)
	if c.debug >= 1 {
		fmt.Println("Modbus client ReadInputRegs req: ", req)
	}
	packet, err := RtuEncode(id, req)
	if err != nil {
		return ret, err
	}

	if c.debug >= 9 {
		fmt.Println("Modbus client ReadInputRegs tx: ", HexDump(packet))
	}

	_, err = c.port.Write(packet)
	if err != nil {
		return ret, err
	}

	// FIXME, what is max modbus packet size?
	buf := make([]byte, 200)
	cnt, err := c.port.Read(buf)
	if err != nil {
		return ret, err
	}

	buf = buf[:cnt]

	if c.debug >= 9 {
		fmt.Println("Modbus client ReadInputRegs rx: ", HexDump(buf))
	}

	resp, err := RtuDecode(buf)
	if err != nil {
		return ret, err
	}

	if c.debug >= 1 {
		fmt.Println("Modbus client ReadInputRegs resp: ", resp)
	}

	return resp.RespReadRegs()
}

// WriteSingleReg writes to a single holding register
func (c *Client) WriteSingleReg(id byte, reg, value uint16) error {
	req := WriteSingleReg(reg, value)
	if c.debug >= 1 {
		fmt.Println("Modbus client WriteSingleReg req: ", req)
	}
	packet, err := RtuEncode(id, req)
	if err != nil {
		return err
	}

	if c.debug >= 9 {
		fmt.Println("Modbus client WriteSingleReg tx: ", HexDump(packet))
	}

	_, err = c.port.Write(packet)
	if err != nil {
		return err
	}

	// FIXME, what is max modbus packet size?
	buf := make([]byte, 200)
	cnt, err := c.port.Read(buf)
	if err != nil {
		fmt.Println("CLIFF: WriteSingleReg: read error: ", err)
		return err
	}

	buf = buf[:cnt]

	if c.debug >= 9 {
		fmt.Println("Modbus client WriteSingleReg rx: ", HexDump(buf))
	}

	resp, err := RtuDecode(buf)
	if err != nil {
		return err
	}

	// FIXME, check return code matches what we sent
	if resp.FunctionCode != FuncCodeWriteSingleRegister {
		return errors.New("resp contains wrong function code")
	}

	if len(resp.Data) < 4 {
		return errors.New("not enough data in resp")
	}

	retAdr := binary.BigEndian.Uint16(resp.Data)
	retValue := binary.BigEndian.Uint16(resp.Data[2:])

	if retAdr != reg {
		return errors.New("Write single reg did not return expected address")
	}

	if retValue != value {
		return errors.New("Write single reg did not return expected value")
	}

	if c.debug >= 1 {
		fmt.Println("Modbus client WriteSingleReg resp: ", resp)
	}

	return nil

}
