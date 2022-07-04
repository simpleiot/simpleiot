package client

import (
	"fmt"
	"log"
	"reflect"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// Manager manages a node type, watches for changes, adds/removes instances that get
// added/deleted
type Manager[T any] struct {
	nc        *nats.Conn
	root      string
	nodeType  string
	construct func(*nats.Conn, T) Client

	nodes   []data.NodeEdge
	clients []Client
}

// NewManager takes constructor for a node client and returns a Manager for that client
// The Node Type is inferred from the Go type passed in, so you must name Go client
// Types to manage the node type definitions.
func NewManager[T any](nc *nats.Conn, root string,
	construct func(nc *nats.Conn, config T) Client) *Manager[T] {
	var x T
	nodeType := reflect.TypeOf(x).Name()

	return &Manager[T]{nc: nc, root: root, nodeType: nodeType, construct: construct}
}

// Start node manager. This function looks for children of a certain node type.
// When new nodes are found, the data is decoded into the client type config, and the
// constructor for the node client is called.
func (m *Manager[T]) Start() error {
	children, err := GetNodeChildren(m.nc, m.root, m.nodeType, false, false)

	if err != nil {
		return err
	}

	// create nodes
	fmt.Printf("CLIFF: node children: %+v\n", children)

	for _, n := range children {
		m.nodes = append(m.nodes, n)

		var config T

		err := data.Decode(n, &config)
		if err != nil {
			log.Println("Error decoding node: ", err)
			continue
		}

		client := m.construct(m.nc, config)
		m.clients = append(m.clients, client)
	}

	for _, c := range m.clients {
		c.Run(nil)
	}

	return nil
}
