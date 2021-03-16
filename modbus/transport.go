package modbus

import (
	"io"
)

// Transport defines an interface that various
// transports (RTU, TCP, etc) implement and can
// be passed to clients/servers
type Transport interface {
	io.ReadWriter
	Encode(byte, PDU) ([]byte, error)
	Decode([]byte) (PDU, error)
}
