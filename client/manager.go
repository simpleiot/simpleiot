package client

import (
	"log"
	"reflect"
	"strings"
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
	chScan    chan struct{}
	clientsWG sync.WaitGroup

	// The following state data is protected by the lock Mutex and must be locked
	// before accessing
	clientStates map[string]*clientState[T]
	lock         sync.Mutex

	// subscription to listen for new points
	upSub *nats.Subscription
}

// NewManager takes constructor for a node client and returns a Manager for that client
// The Node Type is inferred from the Go type passed in, so you must name Go client
// Types to manage the node type definitions.
func NewManager[T any](nc *nats.Conn, root string,
	construct func(nc *nats.Conn, config T) Client) *Manager[T] {
	var x T
	nodeType := reflect.TypeOf(x).Name()
	nodeType = strings.ToLower(nodeType[0:1]) + nodeType[1:]

	return &Manager[T]{
		nc:           nc,
		root:         root,
		nodeType:     nodeType,
		construct:    construct,
		stop:         make(chan struct{}),
		chScan:       make(chan struct{}),
		clientStates: make(map[string]*clientState[T]),
	}
}

// Start node manager. This function looks for children of a certain node type.
// When new nodes are found, the data is decoded into the client type config, and the
// constructor for the node client is called. This call blocks until Stop is called.
func (m *Manager[T]) Start() error {
	// TODO: it may make sense at some point to have a special topic
	// for new nodes so that all client managers don't have to listen
	// to all points
	var err error
	m.upSub, err = m.nc.Subscribe("up.none.>", func(msg *nats.Msg) {
		points, err := data.PbDecodePoints(msg.Data)
		if err != nil {
			log.Println("Error decoding points")
			return
		}

		for _, p := range points {
			if p.Type == data.PointTypeNodeType {
				m.chScan <- struct{}{}
			}
		}
	})

	if err != nil {
		return err
	}

	err = m.scan()
	if err != nil {
		log.Println("Error scanning for new nodes: ", err)
	}

done:
	for {
		select {
		case <-m.stop:
			break done
		case <-time.After(time.Minute):
			err := m.scan()
			if err != nil {
				log.Println("Error scanning for new nodes: ", err)
			}
		case <-m.chScan:
			err := m.scan()
			if err != nil {
				log.Println("Error scanning for new nodes: ", err)
			}
		}
	}

	// wait for clients to shut down
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

	return nil
}

// Stop manager. This also stops all registered clients and causes Start to exit.
func (m *Manager[T]) Stop(err error) {
	if m.upSub != nil {
		m.upSub.Unsubscribe()
	}

	m.lock.Lock()
	for _, c := range m.clientStates {
		c.stop(err)
	}
	m.lock.Unlock()

	close(m.stop)
}

func (m *Manager[T]) scan() error {
	children, err := GetNodeChildren(m.nc, m.root, m.nodeType, false, false)

	if err != nil {
		return err
	}

	if len(children) < 0 {
		return nil
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	found := make(map[string]bool)

	// create new nodes
	for _, n := range children {
		key := mapKey(n)
		found[key] = true

		if _, ok := m.clientStates[key]; ok {
			continue
		}

		cs := newClientState(m.nc, m.construct, n)

		m.clientStates[key] = cs
		m.clientsWG.Add(1)

		go func() {
			err := cs.start()

			if err != nil {
				log.Printf("clientState error %v: %v\n", m.nodeType, err)
			}

			m.lock.Lock()
			delete(m.clientStates, key)
			m.lock.Unlock()

			m.clientsWG.Done()

			// always scan when client is stopped as there may have been child nodes added/removed
			// and we simply want to start over
			// FIXME, this may deadlock, and on shutdown, we don't want things rescanning
			m.chScan <- struct{}{}
		}()

	}

	// remove nodes that have been deleted
	for key, client := range m.clientStates {
		if _, ok := found[key]; ok {
			continue
		}

		// bus was deleted so close and clear it
		log.Println("removing node: ", m.clientStates[key].node.ID)
		client.stop(nil)
		delete(m.clientStates, key)
	}

	return nil
}
