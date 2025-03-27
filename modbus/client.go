package modbus

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/simpleiot/simpleiot/test"
)

// Client defines a Modbus client (master)
type Client struct {
	transport Transport
	debug     int
}

// NewClient is used to create a new modbus client
// port must return an entire packet for each Read().
// github.com/simpleiot/simpleiot/respreader is a good
// way to do this.
func NewClient(transport Transport, debug int) *Client {
	return &Client{
		transport: transport,
		debug:     debug,
	}
}

// SetDebugLevel allows you to change debug level on the fly
func (c *Client) SetDebugLevel(debug int) {
	c.debug = debug
}

// Close closes the client transport
func (c *Client) Close() error {
	return c.transport.Close()
}

// ReadCoils is used to read modbus coils
func (c *Client) ReadCoils(id byte, coil, count uint16) ([]bool, error) {
	ret := []bool{}
	req := ReadCoils(coil, count)
	if c.debug >= 1 {
		fmt.Printf("Modbus client Readcoils ID:0x%x req:%v\n", id, req)
	}
	packet, err := c.transport.Encode(id, req)
	if err != nil {
		return ret, err
	}

	if c.debug >= 9 {
		fmt.Println("Modbus client ReadCoils tx: ", test.HexDump(packet))
	}

	_, err = c.transport.Write(packet)
	if err != nil {
		return ret, err
	}

	// FIXME, what is max modbus packet size?
	buf := make([]byte, 200)
	cnt, err := c.transport.Read(buf)
	if err != nil {
		return ret, err
	}

	buf = buf[:cnt]

	if c.debug >= 9 {
		fmt.Println("Modbus client ReadCoils rx: ", test.HexDump(buf))
	}

	_, resp, err := c.transport.Decode(buf)
	if err != nil {
		return ret, err
	}

	if c.debug >= 1 {
		fmt.Printf("Modbus client Readcoils ID:0x%x resp:%v\n", id, resp)
	}

	return resp.RespReadBits()
}

// WriteSingleCoil is used to read modbus coils
func (c *Client) WriteSingleCoil(id byte, coil uint16, v bool) error {
	req := WriteSingleCoil(coil, v)
	if c.debug >= 1 {
		fmt.Printf("Modbus client WriteSingleCoil ID:0x%x req:%v\n", id, req)
	}
	packet, err := c.transport.Encode(id, req)
	if err != nil {
		return fmt.Errorf("RtuEncode error: %w", err)
	}

	if c.debug >= 9 {
		fmt.Println("Modbus client WriteSingleCoil tx: ", test.HexDump(packet))
	}

	_, err = c.transport.Write(packet)
	if err != nil {
		return err
	}

	// FIXME, what is max modbus packet size?
	buf := make([]byte, 200)
	cnt, err := c.transport.Read(buf)
	if err != nil {
		return err
	}

	buf = buf[:cnt]

	if c.debug >= 9 {
		fmt.Println("Modbus client WriteSingleCoil rx: ", test.HexDump(buf))
	}

	_, resp, err := c.transport.Decode(buf)
	if err != nil {
		return fmt.Errorf("RtuDecode error: %w", err)
	}

	if c.debug >= 1 {
		fmt.Printf("Modbus client WriteSingleCoil ID:0x%x resp:%v\n", id, resp)
	}

	if resp.FunctionCode != req.FunctionCode {
		return errors.New("resp contains wrong function code")
	}

	if !bytes.Equal(req.Data, resp.Data) {
		return errors.New("Did not get the correct response data")
	}

	return nil
}

// ReadDiscreteInputs is used to read modbus discrete inputs
func (c *Client) ReadDiscreteInputs(id byte, input, count uint16) ([]bool, error) {
	ret := []bool{}
	req := ReadDiscreteInputs(input, count)
	if c.debug >= 1 {
		fmt.Printf("Modbus client ReadDiscreteInputs ID:0x%x req:%v\n", id, req)
	}
	packet, err := c.transport.Encode(id, req)
	if err != nil {
		return ret, err
	}

	if c.debug >= 9 {
		fmt.Println("Modbus client ReadDiscreteInputs tx: ", test.HexDump(packet))
	}

	_, err = c.transport.Write(packet)
	if err != nil {
		return ret, err
	}

	// FIXME, what is max modbus packet size?
	buf := make([]byte, 200)
	cnt, err := c.transport.Read(buf)
	if err != nil {
		return ret, err
	}

	buf = buf[:cnt]

	if c.debug >= 9 {
		fmt.Println("Modbus client ReadDiscreteInputs rx: ", test.HexDump(buf))
	}

	_, resp, err := c.transport.Decode(buf)
	if err != nil {
		return ret, err
	}

	if c.debug >= 1 {
		fmt.Printf("Modbus client ReadDiscreteInputs ID:0x%x resp:%v\n", id, resp)
	}

	if resp.FunctionCode != req.FunctionCode {
		return []bool{}, errors.New("resp contains wrong function code")
	}

	return resp.RespReadBits()
}

// ReadHoldingRegs is used to read modbus holding regs
func (c *Client) ReadHoldingRegs(id byte, reg, count uint16) ([]uint16, error) {
	ret := []uint16{}
	req := ReadHoldingRegs(reg, count)
	if c.debug >= 1 {
		fmt.Printf("Modbus client ReadHoldingRegs ID:0x%x req:%v\n", id, req)
	}
	packet, err := c.transport.Encode(id, req)
	if err != nil {
		return ret, err
	}

	if c.debug >= 9 {
		fmt.Println("Modbus client ReadHoldingRegs tx: ", test.HexDump(packet))
	}

	_, err = c.transport.Write(packet)
	if err != nil {
		return ret, err
	}

	// FIXME, what is max modbus packet size?
	buf := make([]byte, 200)
	cnt, err := c.transport.Read(buf)
	if err != nil {
		return ret, err
	}

	buf = buf[:cnt]

	if c.debug >= 9 {
		fmt.Println("Modbus client ReadHoldingRegs rx: ", test.HexDump(buf))
	}

	_, resp, err := c.transport.Decode(buf)
	if err != nil {
		return ret, err
	}

	if c.debug >= 1 {
		fmt.Printf("Modbus client ReadHoldingRegs ID:0x%x resp:%v\n", id, resp)
	}

	if resp.FunctionCode != req.FunctionCode {
		return []uint16{}, errors.New("resp contains wrong function code")
	}

	return resp.RespReadRegs()
}

// ReadInputRegs is used to read modbus input regs
func (c *Client) ReadInputRegs(id byte, reg, count uint16) ([]uint16, error) {
	ret := []uint16{}
	req := ReadInputRegs(reg, count)
	if c.debug >= 1 {
		fmt.Printf("Modbus client ReadInputRegs ID:0x%x req:%v\n", id, req)
	}
	packet, err := c.transport.Encode(id, req)
	if err != nil {
		return ret, err
	}

	if c.debug >= 9 {
		fmt.Println("Modbus client ReadInputRegs tx: ", test.HexDump(packet))
	}

	_, err = c.transport.Write(packet)
	if err != nil {
		return ret, err
	}

	// FIXME, what is max modbus packet size?
	buf := make([]byte, 200)
	cnt, err := c.transport.Read(buf)
	if err != nil {
		return ret, err
	}

	buf = buf[:cnt]

	if c.debug >= 9 {
		fmt.Println("Modbus client ReadInputRegs rx: ", test.HexDump(buf))
	}

	_, resp, err := c.transport.Decode(buf)
	if err != nil {
		return ret, err
	}

	if c.debug >= 1 {
		fmt.Printf("Modbus client ReadInputRegs ID:0x%x resp:%v\n", id, resp)
	}

	if resp.FunctionCode != req.FunctionCode {
		return []uint16{}, errors.New("resp contains wrong function code")
	}

	return resp.RespReadRegs()
}

// WriteSingleReg writes to a single holding register
func (c *Client) WriteSingleReg(id byte, reg, value uint16) error {
	req := WriteSingleReg(reg, value)
	if c.debug >= 1 {
		fmt.Printf("Modbus client WriteSingleReg ID:0x%x req:%v\n", id, req)
	}
	packet, err := c.transport.Encode(id, req)
	if err != nil {
		return err
	}

	if c.debug >= 9 {
		fmt.Println("Modbus client WriteSingleReg tx: ", test.HexDump(packet))
	}

	_, err = c.transport.Write(packet)
	if err != nil {
		return err
	}

	// FIXME, what is max modbus packet size?
	buf := make([]byte, 200)
	cnt, err := c.transport.Read(buf)
	if err != nil {
		return err
	}

	buf = buf[:cnt]

	if c.debug >= 9 {
		fmt.Println("Modbus client WriteSingleReg rx: ", test.HexDump(buf))
	}

	_, resp, err := c.transport.Decode(buf)
	if err != nil {
		return err
	}

	if c.debug >= 1 {
		fmt.Printf("Modbus client WriteSingleReg ID:0x%x resp:%v\n", id, resp)
	}

	if resp.FunctionCode != req.FunctionCode {
		return errors.New("resp contains wrong function code")
	}

	if !bytes.Equal(req.Data, resp.Data) {
		return errors.New("Did not get the correct response data")
	}

	return nil
}

// WriteMultipleRegs writes to multiple holding registers
func (c *Client) WriteMultipleRegs(id byte, reg, quantity uint16, values []uint16) error {
	req := WriteMultipleRegs(reg, quantity, values)

	if c.debug >= 1 {
		fmt.Printf("Modbus client WriteMultipleRegs ID:0x%x req:%v\n", id, req)
	}

	packet, err := c.transport.Encode(id, req)
	if err != nil {
		return err
	}

	if c.debug >= 9 {
		fmt.Println("Modbus client WriteMultipleRegs tx: ", test.HexDump(packet))
	}

	_, err = c.transport.Write(packet)
	if err != nil {
		return err
	}

	// FIXME, what is max modbus packet size?
	buf := make([]byte, 200)
	cnt, err := c.transport.Read(buf)
	if err != nil {
		return err
	}

	buf = buf[:cnt]

	if c.debug >= 9 {
		fmt.Println("Modbus client WriteMultipleRegs rx: ", test.HexDump(buf))
	}

	_, resp, err := c.transport.Decode(buf)
	if err != nil {
		return err
	}

	if c.debug >= 1 {
		fmt.Printf("Modbus client WriteMultipleRegs ID:0x%x resp:%v\n", id, resp)
	}

	if resp.FunctionCode != req.FunctionCode {
		return errors.New("resp contains wrong function code")
	}

	// Build the expected response: starting address (2 bytes) + quantity (2 bytes)
	expectedResp := make([]byte, 4)
	binary.BigEndian.PutUint16(expectedResp[0:2], reg)
	binary.BigEndian.PutUint16(expectedResp[2:4], quantity)

	if !bytes.Equal(expectedResp, resp.Data) {
		return errors.New("Did not get the correct response data")
	}

	return nil
}
