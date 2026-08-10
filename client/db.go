package client

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	influxhttp "github.com/influxdata/influxdb-client-go/v2/api/http"
	influxwrite "github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/simpleiot/simpleiot/data"
)

// InfluxMeasurement is the Influx measurement to which all points are written
const InfluxMeasurement = "points"

// Db represents the configuration for a SIOT DB client
type Db struct {
	ID            string   `node:"id"`
	Parent        string   `node:"parent"`
	Description   string   `point:"description"`
	DbType        string   `point:"dbType"`
	URI           string   `point:"uri"`
	Org           string   `point:"org"`
	Bucket        string   `point:"bucket"`
	AuthToken     string   `point:"authToken"`
	TagPointTypes []string `point:"tagPointType"`
}

// victoriaMetrics returns true if this client is configured to write to a
// Victoria Metrics database. Victoria Metrics accepts the InfluxDB line
// protocol, but only stores numeric samples, so string points are skipped and
// the text field is omitted.
func (dbc *DbClient) victoriaMetrics() bool {
	return dbc.config.DbType == data.PointValueVictoriaMetrics
}

// dbURIValid reports whether uri can be used as a database endpoint. The
// InfluxDB write API needs an absolute HTTP URL; anything else (most often
// an empty URI on a Database node that has not been configured yet) would
// fail on every write, so points are discarded until this returns true.
func dbURIValid(uri string) bool {
	u, err := url.Parse(uri)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

// writer returns the current high-rate write API, or nil when no valid
// URI is configured and points should be discarded.
func (dbc *DbClient) writer() api.WriteAPI {
	dbc.apiMu.Lock()
	defer dbc.apiMu.Unlock()
	return dbc.writeAPI
}

// blockingWriter returns the write API used for stream-delivered
// points, or nil when no valid URI is configured.
func (dbc *DbClient) blockingWriter() api.WriteAPIBlocking {
	dbc.apiMu.Lock()
	defer dbc.apiMu.Unlock()
	return dbc.writeAPIBlocking
}

// DbClient is a SIOT database client. It reads points from the store's
// boundary-origin streams with durable consumers (ADR-7), so delivery
// is resumable: points written while this client, the instance, or an
// upstream connection was down are delivered when things come back.
//
// Stream deliveries are written synchronously and acknowledged only
// once the database has accepted them, so an outage in the database
// itself is resumable in the same way: the points stay in the stream
// and are written when it comes back, leaving no gap in the recorded
// history. High-rate points are not stored in streams and still arrive
// over the phrup wire subjects, where the best that can be done is a
// buffered, best-effort write.
type DbClient struct {
	nc            *nats.Conn
	config        Db
	stop          chan struct{}
	newPoints     chan NewPoints
	newEdgePoints chan NewPoints
	chStreamMsgs  chan dbStreamMsg
	upSubHr       *nats.Subscription
	epSub         *nats.Subscription
	nodeCache     nodeCache

	// client and the write APIs are nil until a valid URI is
	// configured, and are replaced when the connection settings change.
	// The high-rate subscription reads them from its own goroutine, so
	// apiMu guards all three.
	apiMu            sync.Mutex
	client           influxdb2.Client
	writeAPI         api.WriteAPI
	writeAPIBlocking api.WriteAPIBlocking

	// pending holds the points and the stream messages they came from
	// between flushes. Only the main loop touches these.
	pendingPoints []*influxwrite.Point
	pendingMsgs   []jetstream.Msg

	// writeFailures counts consecutive failed flushes, and retryAfter
	// is when the next attempt is worth making.
	writeFailures int
	retryAfter    time.Time

	consumers map[string]jetstream.ConsumeContext
	rootID    string

	memberMu    sync.Mutex
	memberCache map[string]bool
}

// dbStreamMsg carries one decoded stream delivery into the main loop,
// which acknowledges it after handing the points to the writer.
type dbStreamMsg struct {
	msg    jetstream.Msg
	nodeID string
	points data.Points
}

// dbStreamScanPeriod is how often the client looks for new streams to
// consume (e.g. a replica appearing when a device is adopted).
const dbStreamScanPeriod = 3 * time.Second

const (
	// dbBatchSize is how many points are written in one request.
	dbBatchSize = 500

	// dbFlushPeriod bounds how long a partial batch waits before being
	// written.
	dbFlushPeriod = time.Second

	// dbWriteTimeout bounds one write request. The main loop blocks for
	// this long at most when the database stops responding mid-request.
	dbWriteTimeout = 20 * time.Second

	// dbMaxRetryDelay caps the delay between attempts while the
	// database is unreachable.
	dbMaxRetryDelay = time.Minute

	// dbAckWait is how long JetStream waits for an acknowledgement
	// before redelivering. It is comfortably longer than one write so
	// an in-flight request is not duplicated underneath us.
	dbAckWait = 2 * time.Minute

	// dbMaxAckPending bounds how many unacknowledged points the client
	// holds. Once reached, JetStream stops delivering and the points
	// wait in the stream, which is where they are safest.
	dbMaxAckPending = 5000
)

// NewDbClient ...
func NewDbClient(nc *nats.Conn, config Db) Client {
	return &DbClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan NewPoints),
		newEdgePoints: make(chan NewPoints),
		chStreamMsgs:  make(chan dbStreamMsg, 64),
		nodeCache:     newNodeCache(config.TagPointTypes, config.Parent),
		consumers:     make(map[string]jetstream.ConsumeContext),
		memberCache:   make(map[string]bool),
	}
}

// Run runs the main logic for this client and blocks until stopped
func (dbc *DbClient) Run() error {
	log.Println("Starting db client:", dbc.config.Description)
	var err error

	// FIXME, we probably want to store edge points too ...

	rootNode, err := GetRootNode(dbc.nc)
	if err != nil {
		return fmt.Errorf("error getting root node: %w", err)
	}
	dbc.rootID = rootNode.ID

	js, err := jetstream.New(dbc.nc)
	if err != nil {
		return fmt.Errorf("error creating JetStream context: %w", err)
	}

	// node moves change which nodes are under our parent and which
	// ancestor tags they inherit; edges change rarely, so drop both the
	// membership cache and the tag cache on any edge point
	dbc.epSub, err = dbc.nc.Subscribe(SubjectEdgeAllPoints(), func(_ *nats.Msg) {
		dbc.memberMu.Lock()
		dbc.memberCache = make(map[string]bool)
		dbc.memberMu.Unlock()
		dbc.nodeCache.Clear()
	})
	if err != nil {
		return fmt.Errorf("subscribing to edge points: %w", err)
	}

	subjectHR := fmt.Sprintf("phrup.%v.*", dbc.config.Parent)
	dbc.upSubHr, err = dbc.nc.Subscribe(subjectHR, func(msg *nats.Msg) {
		writeAPI := dbc.writer()
		if writeAPI == nil {
			// no valid URI configured; discard
			return
		}

		// find node ID for points
		chunks := strings.Split(msg.Subject, ".")
		if len(chunks) != 3 {
			log.Println("rule client up hr sub, malformed subject:", msg.Subject)
			return
		}

		nodeID := chunks[2]

		// Update nodeCache with no points
		err := dbc.nodeCache.Update(dbc.nc, NewPoints{
			ID: nodeID,
		})
		if err != nil {
			log.Printf("error updating cache: %v", err)
		}

		err = data.DecodeSerialHrPayload(msg.Data, func(pt data.Point) {
			tags := map[string]string{
				"type": pt.Type,
				"key":  pt.Key,
			}
			dbc.nodeCache.CopyTags(nodeID, tags)
			p := influxdb2.NewPoint(InfluxMeasurement,
				tags,
				map[string]interface{}{
					"value": pt.Val(),
				},
				pt.Time)
			writeAPI.WritePoint(p)
		})

		if err != nil {
			log.Println("DB: error decoding HR data:", err)
		}
	})

	if err != nil {
		return fmt.Errorf("subscribing to %v: %w", subjectHR, err)
	}

	closeAPI := func() {
		dbc.apiMu.Lock()
		defer dbc.apiMu.Unlock()
		if dbc.client != nil {
			dbc.client.Close()
		}
		dbc.client = nil
		dbc.writeAPI = nil
		dbc.writeAPIBlocking = nil
	}

	setupAPI := func() {
		dbc.apiMu.Lock()
		defer dbc.apiMu.Unlock()

		dbc.writeFailures = 0
		dbc.retryAfter = time.Time{}

		if !dbURIValid(dbc.config.URI) {
			log.Printf("Db client %v: no valid URI configured (%q), discarding points",
				dbc.config.Description, dbc.config.URI)
			dbc.client = nil
			dbc.writeAPI = nil
			dbc.writeAPIBlocking = nil
			return
		}

		log.Println("Setting up Influx API")
		// you can set things like retries, batching, precision, etc in client options.
		dbc.client = influxdb2.NewClientWithOptions(dbc.config.URI,
			dbc.config.AuthToken, influxdb2.DefaultOptions())
		dbc.writeAPI = dbc.client.WriteAPI(dbc.config.Org, dbc.config.Bucket)
		// stream points are batched by the main loop and written here
		// one batch per request, so the library's own retry queue is not
		// involved; the stream holds anything a write did not accept
		dbc.writeAPIBlocking = dbc.client.WriteAPIBlocking(dbc.config.Org,
			dbc.config.Bucket)

		influxErrors := dbc.writeAPI.Errors()

		go func() {
			for err := range influxErrors {
				if err != nil {
					log.Println("Influx write error:", err)
				}

			}
			log.Println("Influxdb write api closed")
		}()
	}

	setupAPI()

	// consume the boundary-origin streams; rescan for streams that
	// appear later (e.g. a replica when a device is adopted)
	dbc.scanStreams(js, true)
	scanTicker := time.NewTicker(dbStreamScanPeriod)
	defer scanTicker.Stop()

	flushTicker := time.NewTicker(dbFlushPeriod)
	defer flushTicker.Stop()

done:
	for {
		select {
		case <-dbc.stop:
			log.Println("Stopping db client:", dbc.config.Description)
			break done

		case <-scanTicker.C:
			dbc.scanStreams(js, false)

		case <-flushTicker.C:
			dbc.flushPending()

		case sm := <-dbc.chStreamMsgs:
			// a tag edit on any node changes the inherited tags of every
			// node beneath it; drop the cache and let entries re-resolve
			// on their next point. This runs before the subtree check
			// because a node outside the subtree can still be an
			// ancestor of one inside it via another DAG path.
			if dbc.nodeCache.hasTagPointType(sm.points) {
				dbc.nodeCache.Clear()
			}

			if dbc.blockingWriter() == nil {
				// no valid URI configured; discard the points and
				// advance the consumer rather than letting them
				// accumulate for a database we cannot reach
				_ = sm.msg.Ack()
				continue
			}

			if !dbc.underParent(sm.nodeID) {
				// outside this client's subtree; acknowledge so it is
				// not redelivered
				_ = sm.msg.Ack()
				continue
			}

			// Update nodeCache if needed, merging the arriving points so
			// description and tag edits refresh an existing entry
			err := dbc.nodeCache.Update(dbc.nc,
				NewPoints{ID: sm.nodeID, Points: sm.points})
			if err != nil {
				log.Printf("error updating cache: %v", err)
			}

			vm := dbc.victoriaMetrics()
			for _, point := range sm.points {
				if vm && !point.Numeric() {
					// Victoria Metrics only stores numeric samples, so
					// there is nothing useful to write for this point.
					continue
				}
				tags := map[string]string{
					"type": point.Type,
					"key":  point.Key,
				}
				dbc.nodeCache.CopyTags(sm.nodeID, tags)
				fields := map[string]interface{}{
					"value": point.Val(),
				}
				if !vm {
					fields["text"] = point.Txt()
				}
				p := influxdb2.NewPoint(InfluxMeasurement,
					tags,
					fields,
					point.Time)
				dbc.pendingPoints = append(dbc.pendingPoints, p)
			}

			// the message is acknowledged by flushPending, once the
			// database has accepted the points it carried
			dbc.pendingMsgs = append(dbc.pendingMsgs, sm.msg)

			if len(dbc.pendingPoints) >= dbBatchSize ||
				len(dbc.pendingMsgs) >= dbBatchSize {
				dbc.flushPending()
			}

		case pts := <-dbc.newPoints:
			err := data.MergePoints(pts.ID, pts.Points, &dbc.config)
			if err != nil {
				log.Println("error merging new points:", err)
			}

			for _, p := range pts.Points {
				switch p.Type {
				case data.PointTypeURI,
					data.PointTypeOrg,
					data.PointTypeBucket,
					data.PointTypeAuthToken:
					// we need to restart the influx write API; settle
					// anything already batched against the current
					// connection first
					dbc.flushPending()
					closeAPI()
					setupAPI()
				case data.PointTypeTagPointType:
					dbc.nodeCache = newNodeCache(dbc.config.TagPointTypes, dbc.config.Parent)
				}
			}

		case pts := <-dbc.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &dbc.config)
			if err != nil {
				log.Println("error merging new points:", err)
			}
		}
	}

	// clean up
	dbc.flushPending()
	for _, cc := range dbc.consumers {
		cc.Stop()
	}
	// let any consumer callback blocked on the channel finish; the
	// timeout covers a sender that was mid-handoff when Stop ran.
	// These points were never written, so return them to the stream
	// for the next run rather than acknowledging them away.
drain:
	for {
		select {
		case sm := <-dbc.chStreamMsgs:
			_ = sm.msg.Nak()
		case <-time.After(200 * time.Millisecond):
			break drain
		}
	}
	_ = dbc.epSub.Unsubscribe()
	_ = dbc.upSubHr.Unsubscribe()
	closeAPI()
	return nil
}

// flushPending writes the batched points and settles the stream
// messages they came from. Messages are acknowledged only after the
// database has accepted the write, so a database that is down or
// unreachable leaves its points in the stream to be redelivered rather
// than losing them, and the recorded history has no gap once it comes
// back.
func (dbc *DbClient) flushPending() {
	if len(dbc.pendingMsgs) == 0 {
		return
	}

	msgs := dbc.pendingMsgs
	points := dbc.pendingPoints
	dbc.pendingMsgs = nil
	dbc.pendingPoints = nil

	ack := func() {
		for _, m := range msgs {
			_ = m.Ack()
		}
	}
	retry := func(delay time.Duration) {
		for _, m := range msgs {
			_ = m.NakWithDelay(delay)
		}
	}

	writeAPI := dbc.blockingWriter()
	if writeAPI == nil || len(points) == 0 {
		// nothing to write: no database is configured, or every point
		// in this batch was filtered out
		ack()
		return
	}

	if now := time.Now(); now.Before(dbc.retryAfter) {
		// the database was down on the last attempt and the backoff has
		// not expired; return these points to the stream without
		// another connection attempt
		retry(dbc.retryAfter.Sub(now))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbWriteTimeout)
	err := writeAPI.WritePoint(ctx, points...)
	cancel()

	switch {
	case err == nil:
		if dbc.writeFailures > 0 {
			log.Printf("Db client %v: database write succeeded after %v failed attempts",
				dbc.config.Description, dbc.writeFailures)
			dbc.writeFailures = 0
			dbc.retryAfter = time.Time{}
		}
		ack()

	case !dbWriteRetryable(err):
		// the database understood the request and rejected it, so the
		// same points would be rejected again; record the loss and move
		// the consumer past them
		log.Printf("Db client %v: dropping %v points the database rejected: %v",
			dbc.config.Description, len(points), err)
		ack()

	default:
		delay := ExpBackoff(dbc.writeFailures, dbMaxRetryDelay)
		dbc.writeFailures++
		dbc.retryAfter = time.Now().Add(delay)
		if dbc.writeFailures == 1 {
			log.Printf("Db client %v: database write failed, holding points in the stream until it recovers: %v",
				dbc.config.Description, err)
		}
		retry(delay)
	}
}

// dbWriteRetryable reports whether a failed write is worth retrying.
// Connection failures, rate limiting, and server-side errors clear on
// their own once the database is healthy again. A request the database
// rejected outright — bad credentials, an unparsable line — would fail
// the same way every time, so those points are dropped instead of
// holding up everything behind them.
func dbWriteRetryable(err error) bool {
	var herr *influxhttp.Error
	if !errors.As(err, &herr) || herr.StatusCode == 0 {
		// could not reach the database at all
		return true
	}
	return herr.StatusCode == http.StatusTooManyRequests ||
		herr.StatusCode == http.StatusRequestTimeout ||
		herr.StatusCode >= 500
}

// scanStreams looks for boundary-origin streams and starts a durable
// consumer on each new one. Streams present when the client first
// starts get DeliverNew (a new db client does not backfill existing
// history); streams that appear later are new — typically a freshly
// adopted device's replica — and get DeliverAll so their initial
// catch-up is captured.
func (dbc *DbClient) scanStreams(js jetstream.JetStream, firstScan bool) {
	ctx := context.Background()

	lister := js.ListStreams(ctx, jetstream.WithStreamListSubject("inst.>"))
	for si := range lister.Info() {
		name := si.Config.Name
		if _, ok := dbc.consumers[name]; ok {
			continue
		}

		s, err := js.Stream(ctx, name)
		if err != nil {
			log.Printf("DB: error getting stream %v: %v", name, err)
			continue
		}

		deliver := jetstream.DeliverAllPolicy
		if firstScan {
			deliver = jetstream.DeliverNewPolicy
		}

		c, err := s.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
			Durable:       "db-" + dbc.config.ID,
			AckPolicy:     jetstream.AckExplicitPolicy,
			DeliverPolicy: deliver,
			// points are redelivered until the database accepts them,
			// so an outage is ridden out rather than lost
			AckWait:       dbAckWait,
			MaxDeliver:    -1,
			MaxAckPending: dbMaxAckPending,
			// node points only; edge points are not stored (yet)
			FilterSubject: "inst.*.*.*.p.>",
		})
		if err != nil {
			log.Printf("DB: error creating consumer on %v: %v", name, err)
			continue
		}

		cc, err := c.Consume(func(m jetstream.Msg) {
			// inst.<boundary>.<origin>.<nodeID>.p.<type>.<key>
			tok := strings.Split(m.Subject(), ".")
			if len(tok) != 7 {
				_ = m.Ack()
				return
			}

			pts, err := data.DecodePoints(m.Data())
			if err != nil {
				log.Printf("DB: error decoding points from %v: %v", m.Subject(), err)
				_ = m.Ack()
				return
			}
			for i := range pts {
				if pts[i].Type == "" {
					pts[i].Type = tok[5]
				}
				if pts[i].Key == "" {
					pts[i].Key = tok[6]
				}
			}

			dbc.chStreamMsgs <- dbStreamMsg{msg: m, nodeID: tok[3], points: pts}
		})
		if err != nil {
			log.Printf("DB: error consuming %v: %v", name, err)
			continue
		}

		log.Printf("DB: consuming stream %v", name)
		dbc.consumers[name] = cc
	}
	if err := lister.Err(); err != nil {
		log.Println("DB: error listing streams:", err)
	}
}

// underParent reports whether a node is in the subtree of this
// client's parent node, walking up undeleted edges. Results are cached;
// the cache is dropped whenever any edge changes.
func (dbc *DbClient) underParent(id string) bool {
	// a db client directly under the instance root sees everything
	if dbc.config.Parent == dbc.rootID {
		return true
	}

	dbc.memberMu.Lock()
	member, ok := dbc.memberCache[id]
	dbc.memberMu.Unlock()
	if ok {
		return member
	}

	member = dbc.walkUp(id, make(map[string]bool))

	dbc.memberMu.Lock()
	dbc.memberCache[id] = member
	dbc.memberMu.Unlock()

	return member
}

func (dbc *DbClient) walkUp(id string, visited map[string]bool) bool {
	if id == dbc.config.Parent {
		return true
	}
	if visited[id] {
		return false
	}
	visited[id] = true

	nodes, err := GetNodes(dbc.nc, "all", id, "", false)
	if err != nil {
		return false
	}
	for _, n := range nodes {
		if n.Parent == "root" {
			continue
		}
		if dbc.walkUp(n.Parent, visited) {
			return true
		}
	}
	return false
}

// Stop sends a signal to the Run function to exit
func (dbc *DbClient) Stop(_ error) {
	close(dbc.stop)
}

// Points is called by the Manager when new points for this
// node are received.
func (dbc *DbClient) Points(nodeID string, points []data.Point) {
	dbc.newPoints <- NewPoints{nodeID, "", points}
}

// EdgePoints is called by the Manager when new edge points for this
// node are received.
func (dbc *DbClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	dbc.newEdgePoints <- NewPoints{nodeID, parentID, points}
}
