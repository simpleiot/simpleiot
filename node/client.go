package node

import "github.com/simpleiot/simpleiot/data"

// Client interface describes methods a simple iot client must implement.
// This is to be kept as simple as possible, and the ClientManager does all
// the heavy lifting of interacting with the rest of the SIOT system.
type Client interface {
	Stop()
	Run(c <-chan data.Point)
}
