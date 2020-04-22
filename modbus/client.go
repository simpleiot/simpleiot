package modbus

import "io"

// Client defines a Modbus client (master)
// TODO ...
type Client struct {
	port io.ReadWriter
}
