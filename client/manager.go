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

// NewManager ...
func NewManager[T any](nc *nats.Conn, root string,
	construct func(nc *nats.Conn, config T) Client) *Manager[T] {
	var x T
	nodeType := reflect.TypeOf(x).Name()

	return &Manager[T]{nc: nc, root: root, nodeType: nodeType, construct: construct}
}

// Start node manager
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
