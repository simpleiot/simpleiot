package client

import "github.com/simpleiot/simpleiot/data"

// Client interface describes methods a simple iot client must implement.
// This is to be kept as simple as possible, and the ClientManager does all
// the heavy lifting of interacting with the rest of the SIOT system.
// Start should block until Stop is called.
// Start MUST return when Stop is called.
type Client interface {
	Start() error
	Stop(error)
	Update([]data.Point)
}
