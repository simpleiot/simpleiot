package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
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
	URI           string   `point:"uri"`
	Org           string   `point:"org"`
	Bucket        string   `point:"bucket"`
	AuthToken     string   `point:"authToken"`
	TagPointTypes []string `point:"tagPointType"`
}

// DbClient is a SIOT database client. It reads points from the store's
// boundary-origin streams with durable consumers (ADR-7), so delivery
// is resumable: points written while this client, the instance, or an
// upstream connection was down are delivered when things come back.
// High-rate points are not stored in streams and still arrive over the
// phrup wire subjects.
type DbClient struct {
	nc            *nats.Conn
	config        Db
	stop          chan struct{}
	newPoints     chan NewPoints
	newEdgePoints chan NewPoints
	chStreamMsgs  chan dbStreamMsg
	upSubHr       *nats.Subscription
	historySub    *nats.Subscription
	epSub         *nats.Subscription
	nodeCache     nodeCache
	client        influxdb2.Client
	writeAPI      api.WriteAPI

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

// NewDbClient ...
func NewDbClient(nc *nats.Conn, config Db) Client {
	return &DbClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan NewPoints),
		newEdgePoints: make(chan NewPoints),
		chStreamMsgs:  make(chan dbStreamMsg, 64),
		nodeCache:     newNodeCache(config.TagPointTypes),
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

	// node moves change which nodes are under our parent; edges change
	// rarely, so drop the whole membership cache on any edge point
	dbc.epSub, err = dbc.nc.Subscribe(SubjectEdgeAllPoints(), func(_ *nats.Msg) {
		dbc.memberMu.Lock()
		dbc.memberCache = make(map[string]bool)
		dbc.memberMu.Unlock()
	})
	if err != nil {
		return fmt.Errorf("subscribing to edge points: %w", err)
	}

	subjectHR := fmt.Sprintf("phrup.%v.*", dbc.config.Parent)
	dbc.upSubHr, err = dbc.nc.Subscribe(subjectHR, func(msg *nats.Msg) {
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
			dbc.writeAPI.WritePoint(p)
		})

		if err != nil {
			log.Println("DB: error decoding HR data:", err)
		}
	})

	if err != nil {
		return fmt.Errorf("subscribing to %v: %w", subjectHR, err)
	}

	subjectHistory := fmt.Sprintf("history.%v", dbc.config.ID)
	dbc.historySub, err = dbc.nc.Subscribe(subjectHistory, func(msg *nats.Msg) {
		query := new(data.HistoryQuery)
		results := new(data.HistoryResults)
		ctx := context.Background()

		// Defer encoding and sending response
		defer func() {
			res, err := json.Marshal(results)
			if err != nil {
				err = msg.Respond([]byte(`{"error":"error encoding response"}`))
				if err != nil {
					log.Printf("error responding to history query: %v", err)
				}
			} else {
				err = msg.Respond(res)
				if err != nil {
					// Try responding via NATS with the error
					results = &data.HistoryResults{
						ErrorMessage: err.Error(),
					}
					res, parseErr := json.Marshal(results)
					if parseErr == nil {
						retryError := msg.Respond(res)
						if retryError == nil {
							// clear original error
							err = nil
						}
					}
				}
				if err != nil {
					log.Printf("error responding to history query: %v", err)
				}
			}
		}()

		// Parse query
		err = json.Unmarshal(msg.Data, query)
		if err != nil {
			results.ErrorMessage = "parsing query: " + err.Error()
			return
		}
		log.Printf("received history query: %+v", query)

		// Execute query
		query.Execute(
			ctx,
			dbc.client.QueryAPI(dbc.config.Org),
			dbc.config.Bucket,
			InfluxMeasurement,
			results,
		)
	})

	if err != nil {
		return fmt.Errorf("subscribing to %v: %w", subjectHistory, err)
	}

	setupAPI := func() {
		log.Println("Setting up Influx API")
		// you can set things like retries, batching, precision, etc in client options.
		dbc.client = influxdb2.NewClientWithOptions(dbc.config.URI,
			dbc.config.AuthToken, influxdb2.DefaultOptions())
		dbc.writeAPI = dbc.client.WriteAPI(dbc.config.Org, dbc.config.Bucket)

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

done:
	for {
		select {
		case <-dbc.stop:
			log.Println("Stopping db client:", dbc.config.Description)
			break done

		case <-scanTicker.C:
			dbc.scanStreams(js, false)

		case sm := <-dbc.chStreamMsgs:
			if !dbc.underParent(sm.nodeID) {
				// outside this client's subtree; acknowledge so it is
				// not redelivered
				_ = sm.msg.Ack()
				continue
			}

			// Update nodeCache if needed
			err := dbc.nodeCache.Update(dbc.nc, NewPoints{ID: sm.nodeID})
			if err != nil {
				log.Printf("error updating cache: %v", err)
			}

			for _, point := range sm.points {
				tags := map[string]string{
					"type": point.Type,
					"key":  point.Key,
				}
				dbc.nodeCache.CopyTags(sm.nodeID, tags)
				p := influxdb2.NewPoint(InfluxMeasurement,
					tags,
					map[string]interface{}{
						"value": point.Val(),
						"text":  point.Txt(),
					},
					point.Time)
				dbc.writeAPI.WritePoint(p)
			}

			// acknowledged once handed to the batching writer; a crash
			// can lose the writer's unflushed batch, but the durable
			// consumer position means nothing else is ever skipped
			_ = sm.msg.Ack()
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
					// we need to restart the influx write API
					dbc.client.Close()
					setupAPI()
				case data.PointTypeTagPointType:
					dbc.nodeCache = newNodeCache(dbc.config.TagPointTypes)
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
	for _, cc := range dbc.consumers {
		cc.Stop()
	}
	// let any consumer callback blocked on the channel finish; the
	// timeout covers a sender that was mid-handoff when Stop ran
drain:
	for {
		select {
		case sm := <-dbc.chStreamMsgs:
			_ = sm.msg.Ack()
		case <-time.After(200 * time.Millisecond):
			break drain
		}
	}
	_ = dbc.epSub.Unsubscribe()
	_ = dbc.upSubHr.Unsubscribe()
	_ = dbc.historySub.Unsubscribe()
	dbc.client.Close()
	return nil
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
