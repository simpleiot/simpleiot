package modbus

import (
	"io"
)

// TransportType defines a modbus transport type
type TransportType string

// define valid transport types
const (
	TransportTypeTCP TransportType = "tcp"
	TransportTypeRTU               = "rtu"
)

// Transport defines an interface that various
// transports (RTU, TCP, etc) implement and can
// be passed to clients/servers
type Transport interface {
	io.ReadWriteCloser
	Encode(byte, PDU) ([]byte, error)
	Decode([]byte) (PDU, error)
	Type() TransportType
}
