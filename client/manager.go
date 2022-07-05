package client

import (
	"log"
	"reflect"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// Manager manages a node type, watches for changes, adds/removes instances that get
// added/deleted
type Manager[T any] struct {
	// initial state
	nc        *nats.Conn
	root      string
	nodeType  string
	construct func(*nats.Conn, T) Client

	// synchronization fields
	stop      chan struct{}
	clientsWG sync.WaitGroup

	// The following state data is protected by the lock Mutex and must be locked
	// before accessing
	lock          sync.Mutex
	nodes         map[string]data.NodeEdge
	clients       map[string]Client
	subscriptions map[string]*nats.Subscription
}

// NewManager takes constructor for a node client and returns a Manager for that client
// The Node Type is inferred from the Go type passed in, so you must name Go client
// Types to manage the node type definitions.
func NewManager[T any](nc *nats.Conn, root string,
	construct func(nc *nats.Conn, config T) Client) *Manager[T] {
	var x T
	nodeType := reflect.TypeOf(x).Name()

	return &Manager[T]{
		nc:            nc,
		root:          root,
		nodeType:      nodeType,
		construct:     construct,
		stop:          make(chan struct{}),
		nodes:         make(map[string]data.NodeEdge),
		clients:       make(map[string]Client),
		subscriptions: make(map[string]*nats.Subscription),
	}
}

// Start node manager. This function looks for children of a certain node type.
// When new nodes are found, the data is decoded into the client type config, and the
// constructor for the node client is called. This call blocks until Stop is called.
func (m *Manager[T]) Start() error {
	children, err := GetNodeChildren(m.nc, m.root, m.nodeType, false, false)

	if err != nil {
		return err
	}

	m.lock.Lock()
	// create nodes
	for _, n := range children {
		m.nodes[n.ID] = n

		var config T

		err := data.Decode(n, &config)
		if err != nil {
			log.Println("Error decoding node: ", err)
			continue
		}

		client := m.construct(m.nc, config)
		m.clients[n.ID] = client

		func(client Client) {
			sub, err := m.nc.Subscribe("node."+n.ID+".points", func(msg *nats.Msg) {
				points, err := data.PbDecodePoints(msg.Data)
				if err != nil {
					// FIXME, send over channel
					log.Println("Error decoding node data: ", err)
					return
				}
				client.Update(points)

			})
			if err != nil {
				log.Println("client manager sub error: ", err)
				return
			}
			m.subscriptions[n.ID] = sub
		}(client)

		go func(client Client) {
			m.clientsWG.Add(1)
			err := client.Start()
			if err != nil {
				log.Println("Node client returned error: ", err)
			}
			m.clientsWG.Done()
		}(client)
	}
	m.lock.Unlock()

	<-m.stop
	return nil
}

// Stop manager. This also stops all registered clients and causes Start to exit.
func (m *Manager[T]) Stop(err error) {
	m.lock.Lock()
	for _, c := range m.clients {
		c.Stop(err)
	}
	m.lock.Unlock()

	clientsDone := make(chan struct{})
	go func() {
		m.clientsWG.Wait()
		close(clientsDone)
	}()

	select {
	case <-clientsDone:
		// all is well
	case <-time.After(time.Second * 5):
		log.Println("BUG: Not all clients shutdown!")
	}

	close(m.stop)
}
