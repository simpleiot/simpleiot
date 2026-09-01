package client

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/simpleiot/simpleiot/data"
)

// Sync represents a sync node config
type Sync struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	URI         string `point:"uri"`
	AuthToken   string `point:"authToken"`
	// PubKey is this instance's device key, shown so it can be enrolled on
	// the upstream. The client maintains it; the key is used whenever no
	// AuthToken is set.
	PubKey string `point:"pubKey"`
	// Error says why the upstream is not connected when the reason is not
	// going to fix itself, such as a refused credential. The client
	// maintains it.
	Error          string `point:"error"`
	Disabled       bool   `point:"disabled"`
	SyncCount      int    `point:"syncCount"`
	SyncCountReset bool   `point:"syncCountReset"`
}

// SyncClient handles a connection to an upstream instance by
// replicating boundary-origin streams (ADR-7 Stage 3):
//
//   - push: this instance's origin stream for its root boundary
//     (inst_<X>_<X>) is copied into a replica stream on the upstream,
//     using a durable consumer on the local stream so a reconnect
//     resumes where it left off.
//   - pull: upstream-origin streams for this instance's boundary
//     (inst_<X>_<o>, o != X — e.g. configuration the upstream wrote
//     for this instance) are copied into local replica streams, using
//     durable consumers on the upstream streams.
//
// The store on each side consumes replica streams, merges tips into
// its caches, and re-broadcasts changes locally (see store/replica.go).
// No instance ever writes remote data into its own origin streams, so
// echo between instances is impossible by construction.
type SyncClient struct {
	nc            *nats.Conn
	config        Sync
	stop          chan struct{}
	newPoints     chan NewPoints
	newEdgePoints chan NewPoints
	chConnected   chan bool
	chClosed      chan struct{}

	rootLocal data.NodeEdge
	ncRemote  *nats.Conn
	sessions  int

	sessionCancel context.CancelFunc
	sessionDone   chan struct{}
}

// syncPullScanPeriod is how often the pull side rescans the upstream
// for new origin streams in this instance's boundary.
const syncPullScanPeriod = 2 * time.Second

// SyncErrorRefused is the sync node's error when the upstream refuses its
// credential: the key is not enrolled, or its credential is disabled.
const SyncErrorRefused = "credential refused by upstream"

// SyncRefusedRetry is how long to wait before connecting again after the
// upstream closed the connection for good, which is what happens when it
// refuses the credential twice. A refused key is not going to start
// working by trying faster. Tests shorten it.
var SyncRefusedRetry = time.Minute

// NewSyncClient constructor
func NewSyncClient(nc *nats.Conn, config Sync) Client {
	return &SyncClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan NewPoints),
		newEdgePoints: make(chan NewPoints),
		chConnected:   make(chan bool),
		chClosed:      make(chan struct{}, 1),
	}
}

// Run the main logic for this client and blocks until stopped
func (up *SyncClient) Run() error {
	var err error
	up.rootLocal, err = GetRootNode(up.nc)
	if err != nil {
		return fmt.Errorf("error getting root node: %v", err)
	}

	connectTimer := time.NewTimer(time.Millisecond * 10)
	connected := false

done:
	for {
		select {
		case <-up.stop:
			log.Println("Stopping sync client:", up.config.Description)
			break done

		case <-connectTimer.C:
			err := up.connect()
			if err != nil {
				log.Printf("Sync connect failure: %v: %v\n",
					up.config.Description, err)
				connectTimer.Reset(30 * time.Second)
			}

		case conn := <-up.chConnected:
			if conn && !connected {
				connected = true
				up.setError("")
				up.startSession()
			} else if !conn && connected {
				connected = false
				up.stopSession()
			}

		case <-up.chClosed:
			// the client gave up reconnecting (the upstream refused
			// it twice, typically because the credential was
			// revoked); keep running standalone and try again later
			connected = false
			up.stopSession()
			if up.ncRemote != nil && errors.Is(up.ncRemote.LastError(), nats.ErrAuthorization) {
				up.setError(SyncErrorRefused)
			}
			up.disconnect()
			log.Printf("Sync %v: upstream closed the connection, retrying in %v\n",
				up.config.Description, SyncRefusedRetry)
			connectTimer.Reset(SyncRefusedRetry)

		case pts := <-up.newPoints:
			err := data.MergePoints(pts.ID, pts.Points, &up.config)
			if err != nil {
				log.Println("error merging new points:", err)
			}

			for _, p := range pts.Points {
				switch p.Type {
				case data.PointTypeURI,
					data.PointTypeAuthToken,
					data.PointTypeDisabled,
					data.PointTypePubKey:
					// we need to restart the sync connection
					connected = false
					up.stopSession()
					up.disconnect()
					connectTimer.Reset(10 * time.Millisecond)
				}
			}

			if up.config.SyncCountReset {
				up.config.SyncCount = 0
				up.config.SyncCountReset = false

				points := data.Points{
					data.NewPointFloat(data.PointTypeSyncCount, "", 0),
					data.NewPointFloat(data.PointTypeSyncCountReset, "", 0),
				}

				err = SendPoints(up.nc, SubjectNodePoints(up.config.ID), points, false)
				if err != nil {
					log.Println("Error resetting sync count:", err)
				}
			}

		case pts := <-up.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &up.config)
			if err != nil {
				log.Println("error merging new edge points:", err)
			}
		}
	}

	up.stopSession()
	up.disconnect()

	return nil
}

// Stop sends a signal to the Run function to exit
func (up *SyncClient) Stop(_ error) {
	close(up.stop)
}

// Points is called by the Manager when new points for this
// node are received.
func (up *SyncClient) Points(nodeID string, points []data.Point) {
	up.newPoints <- NewPoints{nodeID, "", points}
}

// EdgePoints is called by the Manager when new edge points for this
// node are received.
func (up *SyncClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	up.newEdgePoints <- NewPoints{nodeID, parentID, points}
}

func (up *SyncClient) connect() error {
	if up.config.Disabled {
		log.Printf("Sync %v disabled", up.config.Description)
		return nil
	}

	opts := EdgeOptions{
		URI:       up.config.URI,
		AuthToken: up.config.AuthToken,
		NoEcho:    true,
		Connected: func() {
			up.chConnected <- true
			log.Printf("Sync: %v: Remote Connected: %v\n",
				up.config.Description, up.config.URI)
		},
		Disconnected: func() {
			up.chConnected <- false
			log.Printf("Sync: %v: Remote Disconnected\n", up.config.Description)
		},
		Reconnected: func() {
			up.chConnected <- true
			log.Printf("Sync: %v: Remote Reconnected\n", up.config.Description)
		},
		Closed: func() {
			log.Printf("Sync: %v: Remote Closed\n", up.config.Description)
			select {
			case up.chClosed <- struct{}{}:
			default:
			}
		},
	}

	if up.config.AuthToken == "" {
		// no token: connect with this instance's device key
		seed, pubKey, err := GetDeviceKey(up.nc)
		if err != nil {
			log.Printf("Sync %v: no device key, connecting without credentials: %v",
				up.config.Description, err)
		} else {
			opts.NkeySeed = seed
			if pubKey != up.config.PubKey {
				up.config.PubKey = pubKey
				err := SendNodePoint(up.nc, up.config.ID,
					data.NewPointString(data.PointTypePubKey, "", pubKey), false)
				if err != nil {
					log.Println("Error sending sync pubKey:", err)
				}
			}
		}
	}

	var err error
	up.ncRemote, err = EdgeConnect(opts)

	if err != nil {
		return fmt.Errorf("error connecting to upstream NATS: %v", err)
	}

	return nil
}

// setError records why the upstream is not connected, or clears it, on
// the sync node.
func (up *SyncClient) setError(msg string) {
	if up.config.Error == msg {
		return
	}
	up.config.Error = msg
	err := SendNodePoint(up.nc, up.config.ID,
		data.NewPointString(data.PointTypeError, "", msg), false)
	if err != nil {
		log.Println("Error sending sync error:", err)
	}
}

func (up *SyncClient) disconnect() {
	if up.ncRemote != nil {
		up.ncRemote.Close()
		up.ncRemote = nil
	}

	// a close we asked for is not a refusal
	select {
	case <-up.chClosed:
	default:
	}
}

func (up *SyncClient) startSession() {
	if up.sessionDone != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	up.sessionCancel = cancel
	up.sessionDone = done
	up.sessions++

	// report the number of replication sessions started; a healthy
	// connection has a low count
	up.config.SyncCount = up.sessions
	points := data.Points{
		data.NewPointFloat(data.PointTypeSyncCount, "", float64(up.sessions)),
	}
	err := SendPoints(up.nc, SubjectNodePoints(up.config.ID), points, false)
	if err != nil {
		log.Println("Error sending sync count:", err)
	}

	ncRemote := up.ncRemote
	go func() {
		defer close(done)
		err := up.runSession(ctx, ncRemote)
		if err != nil && ctx.Err() == nil {
			log.Printf("Sync %v: session error: %v\n",
				up.config.Description, err)
		}
	}()
}

func (up *SyncClient) stopSession() {
	if up.sessionDone == nil {
		return
	}
	up.sessionCancel()
	<-up.sessionDone
	up.sessionCancel = nil
	up.sessionDone = nil
}

// runSession runs one replication session over a connected upstream.
func (up *SyncClient) runSession(ctx context.Context, ncRemote *nats.Conn) error {
	X := up.rootLocal.ID

	rootRemote, err := GetRootNode(ncRemote)
	if err != nil {
		return fmt.Errorf("error getting upstream root: %v", err)
	}

	// adoption: make sure this instance exists in the upstream tree.
	// The edge lives in the upstream's boundary (its origin streams);
	// a plain (untagged) edge message makes the upstream persist it as
	// its own write. If the upstream deleted us (tombstoned edge), we
	// stay detached — only the upstream can restore the edge.
	nodes, err := GetNodes(ncRemote, "all", X, "", true)
	if err != nil && err != data.ErrDocumentNotFound {
		return fmt.Errorf("error checking upstream for our node: %v", err)
	}
	if len(nodes) == 0 {
		log.Printf("Sync %v: announcing this instance upstream\n",
			up.config.Description)
		err = SendEdgePoints(ncRemote, X, rootRemote.ID, data.Points{
			data.NewPointFloat(data.PointTypeTombstone, "", 0),
			data.NewPointString(data.PointTypeNodeType, "", data.NodeTypeDevice),
		}, true)
		if err != nil {
			return fmt.Errorf("error announcing instance upstream: %v", err)
		}
	}

	jsLocal, err := jetstream.New(up.nc)
	if err != nil {
		return fmt.Errorf("error creating local JetStream context: %v", err)
	}
	jsRemote, err := jetstream.New(ncRemote)
	if err != nil {
		return fmt.Errorf("error creating remote JetStream context: %v", err)
	}

	// push our origin stream for our root boundary upstream
	pushCC, err := runPump(ctx, jsLocal, jsRemote, X, X, rootRemote.ID)
	if err != nil {
		return fmt.Errorf("error starting push replication: %v", err)
	}
	defer pushCC.Stop()

	// pull upstream-origin streams for our boundary; rescan for new
	// ones (e.g. the first time the upstream writes configuration)
	pulls := make(map[string]pumpStopper)
	defer func() {
		for _, cc := range pulls {
			cc.Stop()
		}
	}()

	ticker := time.NewTicker(syncPullScanPeriod)
	defer ticker.Stop()

	for {
		up.scanPulls(ctx, jsLocal, jsRemote, X, pulls)

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// scanPulls discovers upstream-origin streams for our boundary and
// starts a pull pump for each new one. It asks for names rather than
// configurations, which is the one JetStream listing a device credential
// allows (see server/auth.go); every stream in our boundary is named
// inst_<boundary>_<origin>, so the origin is the rest of the name.
func (up *SyncClient) scanPulls(ctx context.Context, jsLocal, jsRemote jetstream.JetStream,
	boundary string, pulls map[string]pumpStopper) {

	lister := jsRemote.StreamNames(ctx,
		jetstream.WithStreamListSubject(fmt.Sprintf("inst.%v.>", boundary)))

	prefix := fmt.Sprintf("inst_%v_", boundary)

	for name := range lister.Name() {
		o, ok := strings.CutPrefix(name, prefix)
		if !ok || o == "" || o == boundary {
			continue
		}
		if _, running := pulls[name]; running {
			continue
		}

		cc, err := runPump(ctx, jsRemote, jsLocal, boundary, o, boundary)
		if err != nil {
			log.Printf("Sync %v: error starting pull replication %v: %v\n",
				up.config.Description, name, err)
			continue
		}

		log.Printf("Sync %v: replicating %v from upstream\n",
			up.config.Description, name)
		pulls[name] = cc
	}
	if err := lister.Err(); err != nil && ctx.Err() == nil {
		log.Printf("Sync %v: error listing upstream streams: %v\n",
			up.config.Description, err)
	}
}

// StreamBoundaryOrigin extracts the boundary and origin IDs from a
// boundary-origin stream's capture subject ("inst.<b>.<o>.>").
func StreamBoundaryOrigin(cfg jetstream.StreamConfig) (boundary, origin string, ok bool) {
	if len(cfg.Subjects) != 1 {
		return "", "", false
	}
	tok := strings.Split(cfg.Subjects[0], ".")
	if len(tok) != 4 || tok[0] != "inst" || tok[3] != ">" {
		return "", "", false
	}
	return tok[1], tok[2], true
}

// pumpWindowSize is how many messages a pump sends before waiting for the
// receiving server to confirm them. Windowing is what makes a first sync of a
// large stream practical: the round trips overlap instead of running one at a
// time. It stays well under the JetStream client's default limit on
// outstanding async publishes.
const pumpWindowSize = 256

// pumpRetryPeriod is how long a pump waits before resending a window the
// receiving side did not accept.
const pumpRetryPeriod = 5 * time.Second

// pumpMsg is the part of a JetStream message a pump uses. It is an interface
// so the window logic can be exercised without a server.
type pumpMsg interface {
	Subject() string
	Data() []byte
	Ack() error
}

// asyncPublisher is the part of a JetStream context a pump uses.
type asyncPublisher interface {
	PublishAsync(subject string, payload []byte,
		opts ...jetstream.PublishOpt) (jetstream.PubAckFuture, error)
}

// pumpStopper shuts a running pump down.
type pumpStopper interface {
	Stop()
}

// sendWindow publishes every message in a window, waits for the receiving
// server to confirm all of them, and acknowledges the source messages only
// once every publish has succeeded. Acknowledging none of them on failure is
// what keeps ordering intact: the durable consumer redelivers the whole window
// in the order it was stored, so a resend can never place an older point after
// a newer one. That matters because the receiving store reads the last message
// on a subject as that subject's current value.
func sendWindow(ctx context.Context, dst asyncPublisher, window []pumpMsg) error {
	if len(window) == 0 {
		return nil
	}

	futures := make([]jetstream.PubAckFuture, 0, len(window))
	for _, m := range window {
		f, err := dst.PublishAsync(m.Subject(), m.Data())
		if err != nil {
			return fmt.Errorf("error publishing %v: %w", m.Subject(), err)
		}
		futures = append(futures, f)
	}

	// every message is in flight, so waiting on the futures in turn costs
	// about one round trip for the window rather than one per message
	for i, f := range futures {
		select {
		case err := <-f.Err():
			return fmt.Errorf("error replicating %v: %w", window[i].Subject(), err)
		case <-f.Ok():
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	for _, m := range window {
		if err := m.Ack(); err != nil {
			return fmt.Errorf("error acking %v: %w", m.Subject(), err)
		}
	}

	return nil
}

// runPump copies messages from a boundary-origin stream on src into the
// same-named replica stream on dst, preserving subjects. A durable
// consumer on src (named for the receiving instance) makes the copy
// resumable: after a disconnect, only unacknowledged messages are
// redelivered. Messages move in windows and are acknowledged only after dst
// confirms every write in the window; a window that fails is resent rather
// than skipped, so the receiving stream sees each subject in source order.
func runPump(ctx context.Context, src, dst jetstream.JetStream,
	boundary, origin, durableFor string) (pumpStopper, error) {

	name := fmt.Sprintf("inst_%v_%v", boundary, origin)

	// make sure the replica stream exists on the receiving side —
	// create only, never update: the receiving instance's store owns
	// stream configuration (retention etc.) and applies its policy
	// when it discovers the stream
	_, err := dst.Stream(ctx, name)
	if errors.Is(err, jetstream.ErrStreamNotFound) {
		_, err = dst.CreateStream(ctx, jetstream.StreamConfig{
			Name:     name,
			Subjects: []string{fmt.Sprintf("inst.%v.%v.>", boundary, origin)},
		})
		if err != nil && !errors.Is(err, jetstream.ErrStreamNameAlreadyInUse) {
			return nil, fmt.Errorf("error creating replica stream %v: %v", name, err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error checking replica stream %v: %v", name, err)
	}

	s, err := src.Stream(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("error getting source stream %v: %v", name, err)
	}

	c, err := s.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:   "sync-" + durableFor,
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating sync consumer on %v: %v", name, err)
	}

	it, err := c.Messages(jetstream.PullMaxMessages(pumpWindowSize))
	if err != nil {
		return nil, fmt.Errorf("error iterating %v: %v", name, err)
	}

	go func() {
		for {
			window, err := fillWindow(it)
			if err != nil {
				// the iterator was stopped, or the session ended
				return
			}

			// resend until it lands: moving on would let a later
			// message overtake this window on the receiving side
			for {
				err := sendWindow(ctx, dst, window)
				if err == nil {
					break
				}
				if ctx.Err() != nil {
					return
				}
				log.Printf("Sync: error replicating %v, retrying: %v\n", name, err)
				select {
				case <-ctx.Done():
					return
				case <-time.After(pumpRetryPeriod):
				}
			}
		}
	}()

	return it, nil
}

// fillWindow collects up to a full window from the iterator, returning early
// once the source has nothing further waiting so a caught-up pump does not sit
// on a partial window.
func fillWindow(it jetstream.MessagesContext) ([]pumpMsg, error) {
	window := make([]pumpMsg, 0, pumpWindowSize)

	for len(window) < pumpWindowSize {
		msg, err := it.Next()
		if err != nil {
			return nil, err
		}
		window = append(window, msg)

		if meta, err := msg.Metadata(); err == nil && meta.NumPending == 0 {
			break
		}
	}

	return window, nil
}
