package modbus

import (
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

// ReadHoldingRegs is used to read modbus coils
func (c *Client) ReadHoldingRegs(id byte, coil, count uint16) ([]uint16, error) {
	ret := []uint16{}
	req := ReadHoldingRegs(coil, count)
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
