package client

import (
	"github.com/simpleiot/simpleiot/data"
)

// Client interface describes methods a Simple Iot client must implement.
// This is to be kept as simple as possible, and the ClientManager does all
// the heavy lifting of interacting with the rest of the SIOT system.
// Start should block until Stop is called.
// Start MUST return when Stop is called.
// Stop does not block -- wait until Start returns if you need to know the client
// is stopped.
// Points and EdgePoints are called when there are updates to the client node.
type Client interface {
	Start() error
	Stop(error)
	Points(string, []data.Point)
	EdgePoints(string, string, []data.Point)
}
