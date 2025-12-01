//go:build darwin || windows

package client

import (
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// NetworkManagerClient is a SimpleIoT client that manages network interfaces
// and their connections using NetworkManager via DBus
type NetworkManagerClient struct {
}

// NetworkManager client configuration
type NetworkManager struct {
}

// NewNetworkManagerClient returns a new NetworkManagerClient using its
// configuration read from the Client Manager
func NewNetworkManagerClient(nc *nats.Conn, config NetworkManager) Client {
	// TODO: Ensure only one NetworkManager client exists
	return &NetworkManagerClient{}
}

// Run starts the NetworkManager Client
func (c *NetworkManagerClient) Run() error {
	return fmt.Errorf("Error: Network manager not supported on this platform.")
}

// Stop stops the NetworkManager Client
func (c *NetworkManagerClient) Stop(error) {
}

// Points is called when the client's node points are updated
func (c *NetworkManagerClient) Points(_ string, _ []data.Point) {
}

// EdgePoints is called when the client's node edge points are updated
func (c *NetworkManagerClient) EdgePoints(
	_ string, _ string, _ []data.Point,
) {
}
