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

// TransportClientServer defines if transport is being used for a client or server
type TransportClientServer string

// define valid client server types
const (
	TransportClient TransportClientServer = "client"
	TransportServer                       = "server"
)

// Transport defines an interface that various
// transports (RTU, TCP, etc) implement and can
// be passed to clients/servers
type Transport interface {
	io.ReadWriteCloser
	Encode(byte, PDU) ([]byte, error)
	Decode([]byte) (byte, PDU, error)
	Type() TransportType
}
