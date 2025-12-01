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
	nc          *nats.Conn
	root        string
	nodeType    string
	parentTypes []string
	construct   func(*nats.Conn, T) Client

	// synchronization fields
	stop        chan struct{}
	chScan      chan struct{}
	chAction    chan func()
	chCSStopped chan string
	chDeleteCS  chan string

	// keep track of clients
	clientStates map[string]*clientState[T]
	clientUpSub  map[string]*nats.Subscription

	// subscription to listen for new points
	upSub *nats.Subscription
}

// NewManager takes constructor for a node client and returns a Manager for that client
// The Node Type is inferred from the Go type passed in, so you must name Go client
// Types to manage the node type definitions. The manager recursively finds nodes
// that are children of group nodes and the node types found in parentTypes.
func NewManager[T any](nc *nats.Conn,
	construct func(nc *nats.Conn, config T) Client, parentTypes []string) *Manager[T] {
	var x T
	nodeType := data.ToCamelCase(reflect.TypeOf(x).Name())

	return &Manager[T]{
		nc:           nc,
		nodeType:     nodeType,
		parentTypes:  append(parentTypes, data.NodeTypeGroup),
		construct:    construct,
		stop:         make(chan struct{}),
		chScan:       make(chan struct{}),
		chAction:     make(chan func()),
		chCSStopped:  make(chan string),
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
				m.chScan <- struct{}{}
			}
		}
	})

	if err != nil {
		return err
	}

	err = m.scan(m.root)
	if err != nil {
		log.Println("Error scanning for new nodes:", err)
	}

	shutdownTimer := time.NewTimer(time.Hour)
	shutdownTimer.Stop()

	restartTimer := time.NewTimer(time.Hour)
	restartTimer.Stop()

	stopping := false

	scan := func() {
		if stopping {
			return
		}

		err := m.scan(m.root)
		if err != nil {
			log.Println("Error scanning for new nodes:", err)
		}
	}

done:
	for {
		select {
		case <-m.stop:
			stopping = true
			_ = m.upSub.Unsubscribe()
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
		case key := <-m.chCSStopped:
			// TODO: the following can be used to wait until all messages
			// have been drained, but have not been able to get this to
			// work reliably without deadlocking
			err = m.clientUpSub[key].Drain()
			if err != nil {
				log.Println("Error unsubscribing subscription:", err)
			}
			start := time.Now()
			for !m.clientUpSub[key].IsValid() && time.Since(start) <= time.Second*1 {
				if time.Since(start) > time.Second*1 {
					log.Println("Error: timeout waiting for subscription to drain:", key)
					break
				}
				time.Sleep(10 * time.Millisecond)
			}

			m.chDeleteCS <- key
		case key := <-m.chDeleteCS:
			err = m.clientUpSub[key].Unsubscribe()
			if err != nil {
				log.Println("Error unsubscribing subscription:", err)
			}
			delete(m.clientUpSub, key)
			// client state must be deleted after the subscription is stopped
			// as the subscription uses it
			delete(m.clientStates, key)

			if stopping {
				if len(m.clientStates) <= 0 {
					break done
				}
			} else {
				// client may have exited itself due to child
				// node changes so scan to re-initialize it again
				scan()
			}
		case <-shutdownTimer.C:
			// TODO: should we return an error here?
			log.Println("BUG: Client manager: not all clients shutdown for node type:", m.nodeType)
			for _, v := range m.clientStates {
				log.Println("Client stuck for node:", v.node.ID)
			}
			break done
		}
	}

	return nil
}

// Stop manager. This also stops all registered clients and causes Start to exit.
func (m *Manager[T]) Stop(_ error) {
	close(m.stop)
}

func (m *Manager[T]) scanHelper(id string, nodes []data.NodeEdge) ([]data.NodeEdge, error) {
	children, err := GetNodes(m.nc, id, "all", m.nodeType, false)
	if err != nil {
		return nil, err
	}

	nodes = append(nodes, children...)

	// recurse into any nodes that may have children
	for _, parentType := range m.parentTypes {
		parentNodes, err := GetNodes(m.nc, id, "all", parentType, false)
		if err != nil {
			return []data.NodeEdge{}, err
		}
		for _, p := range parentNodes {
			c, err := m.scanHelper(p.ID, nodes)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, c...)
		}
	}

	return nodes, nil
}

func (m *Manager[T]) scan(id string) error {
	nodes, err := m.scanHelper(id, []data.NodeEdge{})
	if err != nil {
		return err
	}

	if len(nodes) == 0 {
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

		// Need to create a new client
		cs, err := newClientState(m.nc, m.construct, n)

		if err != nil {
			log.Printf("Error starting client %v: %v", n, err)
		}

		go func() {
			err := cs.run()

			if err != nil {
				log.Printf("clientState error %v: %v\n", m.nodeType, err)
			}

			m.chDeleteCS <- key
		}()

		m.clientStates[key] = cs

		// Set up subscriptions
		subject := fmt.Sprintf("up.%v.>", cs.node.ID)

		m.clientUpSub[key], err = cs.nc.Subscribe(subject, func(msg *nats.Msg) {
			points, err := data.PbDecodePoints(msg.Data)
			if err != nil {
				log.Println("Error decoding points")
				return
			}

			// find node ID for points
			chunks := strings.Split(msg.Subject, ".")

			if len(chunks) != 3 && len(chunks) != 4 {
				log.Println("up subject malformed:", msg.Subject)
				return
			}

			nodeID := chunks[2]

			if len(chunks) == 3 {
				// process node points

				// only filter node points for now. The Shelly client broke badly
				// when we applied the below filtering to edge points as well,
				// probably because the tombstone edge points were filtered.
				// We may optimize this later if we make extensive use of edge
				// points.
				for _, p := range points {
					if p.Origin == "" && nodeID == cs.node.ID {
						// if this point came from the owning client, it already knows about it
						return
					}

					if p.Origin == cs.node.ID {
						// if this client sent this point, it already knows about it
						return
					}
				}

				cs.client.Points(nodeID, points)
			} else if len(chunks) == 4 {
				// process edge points
				parentID := chunks[3]
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
								log.Println("Client state error getting nodes:", err)
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
								log.Println("Client state error getting nodes:", err)
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
				if cs.client == nil {
					log.Fatal("Client is nil: ", cs.node.ID)
				}
				cs.client.EdgePoints(chunks[2], chunks[3], points)
			}
		})

		if err != nil {
			return err
		}

	}

	// remove nodes that have been deleted
	for key, client := range m.clientStates {
		if _, ok := found[key]; ok {
			continue
		}

		// bus was deleted so close and clear it
		log.Println("removing client node:", m.clientStates[key].node.ID)
		client.stop(nil)
	}

	return nil
}

func mapKey(node data.NodeEdge) string {
	return node.Parent + "-" + node.ID
}
