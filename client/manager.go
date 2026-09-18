package client

import (
	"errors"
	"fmt"
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

	// faults remembers, per client key, the errors the manager itself
	// recorded on a node, so a node whose points do not decode is written
	// once rather than at every scan, a client that keeps crashing restarts
	// with backoff, and a client that recovers has its error cleared. It is
	// written from client goroutines as well as the manager loop.
	faultsMu sync.Mutex
	faults   map[string]*clientFault

	// subscription to listen for new points
	upSub *nats.Subscription
}

// clientFault is what the manager knows about a client that failed
type clientFault struct {
	// crashes is the number of consecutive panics
	crashes int
	// err is the text last written to the node's error point
	err string
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
		faults:       make(map[string]*clientFault),
	}
}

// Run node manager. This function looks for children of a certain node type.
// When new nodes are found, the data is decoded into the client type config, and the
// constructor for the node client is called. This call blocks until Stop is called.
func (m *Manager[T]) Run() error {
	err := m.updateRoot()
	if err != nil {
		return err
	}

	what := "client manager " + m.nodeType

	// TODO: it may make sense at some point to have a special topic
	// for new nodes so that all client managers don't have to listen
	// to all points
	m.upSub, err = m.nc.Subscribe("up.root.>", func(msg *nats.Msg) {
		_ = runRecovered(what+" root points", func() error {
			points, err := data.DecodePoints(msg.Data)
			if err != nil {
				log.Println("Error decoding points")
				return nil
			}

			for _, p := range points {
				switch p.Type {
				case data.PointTypeNodeType, data.PointTypePrimary,
					data.PointTypeMirror:
					// a role change decides whether a client runs
					// here, so pick it up now rather than on the
					// next periodic scan
					m.chScan <- struct{}{}
				}
			}
			return nil
		})
	})

	if err != nil {
		return err
	}

	err = runRecovered(what+" scan", m.scan)
	if err != nil {
		log.Println("Error scanning for new nodes:", err)
	}

	// the loop is started again after a panic, with backoff. Stop closes
	// m.stop, so a loop restarted while stopping shuts the clients down
	// rather than waiting out the delay.
	for attempts := 0; ; attempts++ {
		start := time.Now()
		err := runRecovered(what, m.loop)

		var pe *panicError
		if !errors.As(err, &pe) {
			return err
		}

		if time.Since(start) > clientRestartMax {
			attempts = 0
		}

		delay := ExpBackoff(attempts, clientRestartMax)
		log.Printf("%v: restarting in %v", what, delay.Round(time.Millisecond))

		select {
		case <-time.After(delay):
		case <-m.stop:
		}
	}
}

// loop services the manager until Stop is called and every client has
// stopped
func (m *Manager[T]) loop() error {
	shutdownTimer := time.NewTimer(time.Hour)
	shutdownTimer.Stop()

	stopping := false

	scan := func() {
		if stopping {
			return
		}

		err := m.scan()
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
					c.stop(nil)
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
			err := m.clientUpSub[key].Drain()
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
			err := m.clientUpSub[key].Unsubscribe()
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

// recordError writes errS to the node's error point, unless it is what the
// manager wrote there last
func (m *Manager[T]) recordError(key, nodeID, errS string) {
	m.faultsMu.Lock()
	defer m.faultsMu.Unlock()

	f := m.faults[key]
	if f == nil {
		f = &clientFault{}
		m.faults[key] = f
	}

	if f.err == errS {
		return
	}

	p := data.NewPointString(data.PointTypeError, "", errS)
	if err := SendNodePoint(m.nc, nodeID, p, false); err != nil {
		log.Printf("Error recording error on node %v: %v", nodeID, err)
		return
	}

	f.err = errS
}

// clearError forgets a client's faults, and clears the error the manager
// recorded on its node if there was one
func (m *Manager[T]) clearError(key, nodeID string) {
	m.faultsMu.Lock()
	defer m.faultsMu.Unlock()

	f := m.faults[key]
	if f == nil {
		return
	}

	if f.err != "" {
		p := data.NewPointString(data.PointTypeError, "", "")
		if err := SendNodePoint(m.nc, nodeID, p, false); err != nil {
			log.Printf("Error clearing error on node %v: %v", nodeID, err)
			return
		}
	}

	delete(m.faults, key)
}

// forgetFault drops what the manager knows about a client, without touching
// the node
func (m *Manager[T]) forgetFault(key string) {
	m.faultsMu.Lock()
	defer m.faultsMu.Unlock()

	delete(m.faults, key)
}

func (m *Manager[T]) crashCount(key string) int {
	m.faultsMu.Lock()
	defer m.faultsMu.Unlock()

	if f := m.faults[key]; f != nil {
		return f.crashes
	}

	return 0
}

func (m *Manager[T]) setCrashCount(key string, crashes int) {
	m.faultsMu.Lock()
	defer m.faultsMu.Unlock()

	f := m.faults[key]
	if f == nil {
		f = &clientFault{}
		m.faults[key] = f
	}

	f.crashes = crashes
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
			// a mirror of a group displays what is under it without
			// being a second place those nodes run, so descending
			// into one would start a second client for every child
			if p.EdgeRole() == data.EdgeRoleMirror {
				continue
			}

			nodes, err = m.scanHelper(p.ID, nodes)
			if err != nil {
				return nil, err
			}
		}
	}

	return nodes, nil
}

// updateRoot refreshes the cached root node ID. The root node can change while
// we are running -- importing a configuration with `-parentID=root` creates a
// new root node and deletes the old one. Managers scan below the root, so a
// stale ID means nodes created by the import are never discovered.
func (m *Manager[T]) updateRoot() error {
	root, err := GetRootNode(m.nc)
	if err != nil {
		return fmt.Errorf("Manager: error getting root node: %v", err)
	}

	m.root = root.ID

	return nil
}

func (m *Manager[T]) scan() error {
	err := m.updateRoot()
	if err != nil {
		return err
	}

	nodes, err := m.scanHelper(m.root, []data.NodeEdge{})
	if err != nil {
		return err
	}

	found := make(map[string]bool)

	// create new nodes
	for _, n := range nodes {
		// A mirror displays a node somewhere else in the tree. Its
		// client runs on the primary edge, so starting one here would
		// mean two clients driving one piece of hardware with no way
		// to coordinate. Leaving the key out of found also stops a
		// client that is already running if its edge becomes a mirror,
		// through the removal pass at the end of this function.
		if n.EdgeRole() == data.EdgeRoleMirror {
			continue
		}

		key := mapKey(n)
		found[key] = true

		if _, ok := m.clientStates[key]; ok {
			continue
		}

		// Need to create a new client
		cs, err := newClientState(m.nc, m.construct, n)

		if err != nil {
			// a node whose points do not decode (a slice point keyed
			// with something other than an index, for one) is left
			// with the error on it until it is fixed; the next scan
			// tries it again
			log.Printf("Error starting client %v %v: %v", m.nodeType, n.ID, err)
			m.recordError(key, n.ID, err.Error())
			continue
		}

		cs.crashes = m.crashCount(key)
		if cs.crashes == 0 {
			// the error the manager recorded, if any, was a decode
			// failure, and the node decodes now
			m.clearError(key, n.ID)
		}

		go m.runClientState(key, cs)

		m.clientStates[key] = cs

		// Set up subscriptions
		subject := fmt.Sprintf("up.%v.>", cs.node.ID)

		m.clientUpSub[key], err = cs.nc.Subscribe(subject, func(msg *nats.Msg) {
			err := runRecovered(cs.what()+" points", func() error {
				m.clientPoints(cs, msg)
				return nil
			})
			if err != nil {
				cs.crash(err)
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
		m.forgetFault(key)
	}

	return nil
}

// runClientState runs a client until it stops, and tells the manager. When
// the client stopped because it panicked, the error is recorded on its node
// and the restart waits with backoff, so a client that panics on a stored
// point does not spin. A client that stays up for clientRestartMax after a
// crash has its error cleared.
func (m *Manager[T]) runClientState(key string, cs *clientState[T]) {
	start := time.Now()

	var healthy *time.Timer
	if cs.crashes > 0 {
		healthy = time.AfterFunc(clientRestartMax, func() {
			m.clearError(key, cs.node.ID)
		})
	}

	err := cs.run()

	if healthy != nil {
		healthy.Stop()
	}

	var pe *panicError
	if errors.As(err, &pe) {
		crashes := cs.crashes + 1
		if time.Since(start) > clientRestartMax {
			crashes = 1
		}

		m.recordError(key, cs.node.ID, pe.Error())
		m.setCrashCount(key, crashes)

		delay := ExpBackoff(crashes-1, clientRestartMax)
		log.Printf("%v: restarting in %v", cs.what(), delay.Round(time.Millisecond))

		select {
		case <-time.After(delay):
		case <-m.stop:
		}
	} else if err != nil {
		log.Printf("clientState error %v: %v\n", m.nodeType, err)
	}

	m.chDeleteCS <- key
}

// clientPoints delivers a message on a client's up subject to the client,
// restarting the client when the node itself was added or deleted
func (m *Manager[T]) clientPoints(cs *clientState[T], msg *nats.Msg) {
	points, err := data.DecodePoints(msg.Data)
	if err != nil {
		log.Println("Error decoding points")
		return
	}

	// find node ID for points
	chunks := strings.Split(msg.Subject, ".")

	// up.<upId>.<nodeId>.<type>.<key> = 5 chunks (node points)
	// up.<upId>.<nodeId>.<parentId>.<type>.<key> = 6 chunks (edge points)
	if len(chunks) != 5 && len(chunks) != 6 {
		log.Println("up subject malformed:", msg.Subject)
		return
	}

	nodeID := chunks[2]

	if len(chunks) == 5 {
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
	} else if len(chunks) == 6 {
		// process edge points
		parentID := chunks[3]
		for _, p := range points {
			switch {
			case p.Type == data.PointTypeTombstone && p.Val() == 1:
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

			case (p.Type == data.PointTypeTombstone && p.Val() == 0) ||
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
}

func mapKey(node data.NodeEdge) string {
	return node.Parent + "-" + node.ID
}
