package client

import (
	"fmt"
	"log"
	"reflect"
	"strings"
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
	stop       chan struct{}
	chScan     chan struct{}
	chAction   chan func()
	chDeleteCS chan string

	// keep track of clients
	clientStates map[string]*clientState[T]
	clientUpSub  map[string]*nats.Subscription

	// subscription to listen for new points
	upSub *nats.Subscription
}

// NewManager takes constructor for a node client and returns a Manager for that client
// The Node Type is inferred from the Go type passed in, so you must name Go client
// Types to manage the node type definitions.
func NewManager[T any](nc *nats.Conn,
	construct func(nc *nats.Conn, config T) Client) *Manager[T] {
	var x T
	nodeType := reflect.TypeOf(x).Name()
	nodeType = strings.ToLower(nodeType[0:1]) + nodeType[1:]

	return &Manager[T]{
		nc:           nc,
		nodeType:     nodeType,
		construct:    construct,
		stop:         make(chan struct{}),
		chScan:       make(chan struct{}),
		chAction:     make(chan func()),
		chDeleteCS:   make(chan string),
		clientStates: make(map[string]*clientState[T]),
		clientUpSub:  make(map[string]*nats.Subscription),
	}
}

// Run node manager. This function looks for children of a certain node type.
// When new nodes are found, the data is decoded into the client type config, and the
// constructor for the node client is called. This call blocks until Stop is called.
func (m *Manager[T]) Run() error {
	nodes, err := GetNodes(m.nc, "root", "all", "", false)
	if err != nil {
		return fmt.Errorf("Manager: Error getting root node: %v", err)
	}

	if len(nodes) < 1 {
		return fmt.Errorf("Manager: Error no root node")
	}

	m.root = nodes[0].ID

	// TODO: it may make sense at some point to have a special topic
	// for new nodes so that all client managers don't have to listen
	// to all points
	m.upSub, err = m.nc.Subscribe("up.root.>", func(msg *nats.Msg) {
		points, err := data.PbDecodePoints(msg.Data)
		if err != nil {
			log.Println("Error decoding points")
			return
		}

		for _, p := range points {
			if p.Type == data.PointTypeNodeType {
				// FIXME: we have a race condition here where the edge points
				// are sent first, which triggers this, but the node points
				// are still coming in. For now delay a bit to give node
				// points time to come in. Long term we need to sequence
				// things so this always works
				m.chScan <- struct{}{}
			}
		}
	})

	if err != nil {
		return err
	}

	err = m.scan(m.root)
	if err != nil {
		log.Println("Error scanning for new nodes: ", err)
	}

	shutdownTimer := time.NewTimer(time.Hour)
	shutdownTimer.Stop()

	stopping := false

	scan := func() {
		if stopping {
			return
		}

		err := m.scan(m.root)
		if err != nil {
			log.Println("Error scanning for new nodes: ", err)
		}
	}

done:
	for {
		select {
		case <-m.stop:
			stopping = true
			m.upSub.Unsubscribe()
			if len(m.clientStates) > 0 {
				for _, c := range m.clientStates {
					c.stop(err)
				}
				shutdownTimer.Reset(time.Second * 5)
			} else {
				break done
			}
		case f := <-m.chAction:
			f()
		case <-time.After(time.Minute):
			scan()
		case <-m.chScan:
			scan()
		case key := <-m.chDeleteCS:
			delete(m.clientStates, key)
			m.clientUpSub[key].Unsubscribe()
			delete(m.clientUpSub, key)
			if stopping {
				if len(m.clientStates) <= 0 {
					break done
				}
			} else {
				// client may have exitted itself due to child
				// node changes so scan to re-initialize it again
				scan()
			}
		case <-shutdownTimer.C:
			// FIXME: should we return an error here?
			log.Println("BUG: Client manager: not all clients shutdown for node type: ", m.nodeType)
			for _, v := range m.clientStates {
				log.Println("Client stuck for node: ", v.node.ID)
			}
			break done
		}
	}

	return nil
}

// Stop manager. This also stops all registered clients and causes Start to exit.
func (m *Manager[T]) Stop(err error) {
	close(m.stop)
}

func (m *Manager[T]) scanHelper(id string, nodes []data.NodeEdge) ([]data.NodeEdge, error) {
	children, err := GetNodes(m.nc, id, "all", m.nodeType, false)
	if err != nil {
		return nil, err
	}

	nodes = append(nodes, children...)

	// recurse into any groups
	groups, err := GetNodes(m.nc, id, "all", data.NodeTypeGroup, false)
	for _, g := range groups {
		c, err := m.scanHelper(g.ID, nodes)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, c...)
	}

	// FIXME: we need a better way of identifying nodes than
	// can function as "groups" that may have children that require
	// clients.
	shelly, err := GetNodes(m.nc, id, "all", data.NodeTypeShelly, false)
	for _, g := range shelly {
		c, err := m.scanHelper(g.ID, nodes)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, c...)
	}

	return nodes, nil
}

func (m *Manager[T]) scan(id string) error {
	nodes, err := m.scanHelper(id, []data.NodeEdge{})
	if err != nil {
		return err
	}

	if len(nodes) < 0 {
		return nil
	}

	found := make(map[string]bool)

	// create new nodes
	for _, n := range nodes {
		key := mapKey(n)
		found[key] = true

		if _, ok := m.clientStates[key]; ok {
			continue
		}

		cs := newClientState(m.nc, m.construct, n)

		m.clientStates[key] = cs

		// Set up subscriptions
		subject := fmt.Sprintf("up.%v.>", cs.node.ID)

		m.clientUpSub[key], err = cs.nc.Subscribe(subject, func(msg *nats.Msg) {
			points, err := data.PbDecodePoints(msg.Data)
			if err != nil {
				log.Println("Error decoding points")
				return
			}
			for _, p := range points {
				if p.Origin == "" {
					// point came from the owning node, we already know about it
					return
				}
			}

			// find node ID for points
			chunks := strings.Split(msg.Subject, ".")
			if len(chunks) == 3 {
				cs.client.Points(chunks[2], points)
			} else if len(chunks) == 4 {
				nodeID := chunks[2]
				parentID := chunks[3]
				// edge points
				for _, p := range points {
					switch {
					case p.Type == data.PointTypeTombstone && p.Value == 1:
						// node was deleted, make sure we don't see it in DB
						// before restarting client
						start := time.Now()
						for {
							if time.Since(start) > time.Second*5 {
								log.Println("Client state timeout getting nodes")
								cs.stop(nil)
								return
							}
							nodes, err := GetNodes(cs.nc, parentID, nodeID, "", false)
							if err != nil {
								log.Println("Client state error getting nodes: ", err)
								cs.stop(nil)
								return
							}
							if len(nodes) == 0 {
								// confirmed the node was deleted
								cs.stop(nil)
								return
							}
							time.Sleep(time.Millisecond * 10)
						}

					case (p.Type == data.PointTypeTombstone && p.Value == 0) ||
						p.Type == data.PointTypeNodeType:
						// node was created or undeleted, make sure we see it in DB
						// before restarting client
						start := time.Now()
						for {
							if time.Since(start) > time.Second*5 {
								log.Println("Client state timeout getting nodes")
								cs.stop(nil)
								return
							}
							nodes, err := GetNodes(cs.nc, parentID, nodeID, "", false)
							if err != nil {
								log.Println("Client state error getting nodes: ", err)
								cs.stop(nil)
								return
							}
							if len(nodes) > 0 {
								// confirmed the node was added
								cs.stop(nil)
								return
							}
							time.Sleep(time.Millisecond * 10)
						}
					}
				}

				// send edge points to client
				cs.client.EdgePoints(chunks[2], chunks[3], points)
			} else {
				log.Println("up subject malformed: ", msg.Subject)
				return
			}

		})

		if err != nil {
			return err
		}

		go func() {
			err := cs.run()

			if err != nil {
				log.Printf("clientState error %v: %v\n", m.nodeType, err)
			}

			m.chDeleteCS <- key
		}()
	}

	// remove nodes that have been deleted
	for key, client := range m.clientStates {
		if _, ok := found[key]; ok {
			continue
		}

		// bus was deleted so close and clear it
		log.Println("removing client node: ", m.clientStates[key].node.ID)
		client.stop(nil)
	}

	return nil
}
