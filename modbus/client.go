package modbus

import (
	"io"
)

// Client defines a Modbus client (master)
type Client struct {
	port io.ReadWriter
}

// NewClient is used to create a new modbus client
// port must return an entire packet for each Read().
// github.com/simpleiot/simpleiot/respreader is a good
// way to do this.
func NewClient(port io.ReadWriter) *Client {
	return &Client{
		port: port,
	}
}

// ReadCoils is used to read modbus coils
func (c *Client) ReadCoils(id byte, coil, count uint16) ([]bool, error) {
	ret := []bool{}
	packet, err := RtuEncode(id, ReadCoils(coil, count))
	if err != nil {
		return ret, err
	}

	_, err = c.port.Write(packet)
	if err != nil {
		return ret, err
	}

	// FIXME, what is max modbus packet size?
	buf := make([]byte, 200)
	_, err = c.port.Read(buf)
	if err != nil {
		return ret, err
	}

	resp, err := RtuDecode(buf)
	if err != nil {
		return ret, err
	}

	return resp.RespReadBits()
}
