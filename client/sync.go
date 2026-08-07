package client

import (
	"context"
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
	ID             string `node:"id"`
	Parent         string `node:"parent"`
	Description    string `point:"description"`
	URI            string `point:"uri"`
	AuthToken      string `point:"authToken"`
	Period         int    `point:"period"`
	Disabled       bool   `point:"disabled"`
	SyncCount      int    `point:"syncCount"`
	SyncCountReset bool   `point:"syncCountReset"`
}

// SyncClient handles a connection to an upstream instance by
// replicating boundary-origin streams (ADR-7 Stage 3):
//
//   - push: this instance's origin stream for its root boundary
//     (inst-<X>-<X>) is copied into a replica stream on the upstream,
//     using a durable consumer on the local stream so a reconnect
//     resumes where it left off.
//   - pull: upstream-origin streams for this instance's boundary
//     (inst-<X>-<o>, o != X — e.g. configuration the upstream wrote
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

	rootLocal data.NodeEdge
	ncRemote  *nats.Conn
	sessions  int

	sessionCancel context.CancelFunc
	sessionDone   chan struct{}
}

// syncPullScanPeriod is how often the pull side rescans the upstream
// for new origin streams in this instance's boundary.
const syncPullScanPeriod = 2 * time.Second

// NewSyncClient constructor
func NewSyncClient(nc *nats.Conn, config Sync) Client {
	return &SyncClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan NewPoints),
		newEdgePoints: make(chan NewPoints),
		chConnected:   make(chan bool),
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
				up.startSession()
			} else if !conn && connected {
				connected = false
				up.stopSession()
			}

		case pts := <-up.newPoints:
			err := data.MergePoints(pts.ID, pts.Points, &up.config)
			if err != nil {
				log.Println("error merging new points:", err)
			}

			for _, p := range pts.Points {
				switch p.Type {
				case data.PointTypeURI,
					data.PointTypeAuthToken,
					data.PointTypeDisabled:
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
		},
	}

	var err error
	up.ncRemote, err = EdgeConnect(opts)

	if err != nil {
		return fmt.Errorf("error connecting to upstream NATS: %v", err)
	}

	return nil
}

func (up *SyncClient) disconnect() {
	if up.ncRemote != nil {
		up.ncRemote.Close()
		up.ncRemote = nil
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
	pulls := make(map[string]jetstream.ConsumeContext)
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
// starts a pull pump for each new one.
func (up *SyncClient) scanPulls(ctx context.Context, jsLocal, jsRemote jetstream.JetStream,
	boundary string, pulls map[string]jetstream.ConsumeContext) {

	lister := jsRemote.ListStreams(ctx,
		jetstream.WithStreamListSubject(fmt.Sprintf("inst.%v.>", boundary)))

	for si := range lister.Info() {
		b, o, ok := streamBoundaryOrigin(si.Config)
		if !ok || b != boundary || o == boundary {
			continue
		}
		if _, running := pulls[si.Config.Name]; running {
			continue
		}

		cc, err := runPump(ctx, jsRemote, jsLocal, b, o, boundary)
		if err != nil {
			log.Printf("Sync %v: error starting pull replication %v: %v\n",
				up.config.Description, si.Config.Name, err)
			continue
		}

		log.Printf("Sync %v: replicating %v from upstream\n",
			up.config.Description, si.Config.Name)
		pulls[si.Config.Name] = cc
	}
	if err := lister.Err(); err != nil && ctx.Err() == nil {
		log.Printf("Sync %v: error listing upstream streams: %v\n",
			up.config.Description, err)
	}
}

// streamBoundaryOrigin extracts the boundary and origin IDs from a
// boundary-origin stream's capture subject ("inst.<b>.<o>.>").
func streamBoundaryOrigin(cfg jetstream.StreamConfig) (boundary, origin string, ok bool) {
	if len(cfg.Subjects) != 1 {
		return "", "", false
	}
	tok := strings.Split(cfg.Subjects[0], ".")
	if len(tok) != 4 || tok[0] != "inst" || tok[3] != ">" {
		return "", "", false
	}
	return tok[1], tok[2], true
}

// runPump copies messages from a boundary-origin stream on src into the
// same-named replica stream on dst, preserving subjects. A durable
// consumer on src (named for the receiving instance) makes the copy
// resumable: after a disconnect, only unacknowledged messages are
// redelivered. Messages are acknowledged only after dst confirms the
// write.
func runPump(ctx context.Context, src, dst jetstream.JetStream,
	boundary, origin, durableFor string) (jetstream.ConsumeContext, error) {

	name := fmt.Sprintf("inst-%v-%v", boundary, origin)

	// ensure the replica stream exists on the receiving side
	_, err := dst.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     name,
		Subjects: []string{fmt.Sprintf("inst.%v.%v.>", boundary, origin)},
	})
	if err != nil {
		return nil, fmt.Errorf("error ensuring replica stream %v: %v", name, err)
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

	return c.Consume(func(msg jetstream.Msg) {
		_, err := dst.Publish(ctx, msg.Subject(), msg.Data())
		if err != nil {
			if ctx.Err() == nil {
				log.Printf("Sync: error replicating %v: %v\n", msg.Subject(), err)
			}
			// leave unacknowledged; it will be redelivered
			return
		}
		err = msg.Ack()
		if err != nil && ctx.Err() == nil {
			log.Println("Sync: error acking replicated msg:", err)
		}
	})
}
