package store

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/api"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/internal/pb"
	"google.golang.org/protobuf/proto"
)

var reportMetricsPeriod = time.Minute

// Store implements the SIOT NATS api
type Store struct {
	params        Params
	nc            *nats.Conn
	subscriptions map[string]*nats.Subscription
	db            *DbSqlite
	authorizer    api.Authorizer

	// cycle metrics track how long it takes to handle a point
	metricCycleNodePoint     *client.Metric
	metricCycleNodeEdgePoint *client.Metric
	metricCycleNode          *client.Metric
	metricCycleNodeChildren  *client.Metric

	// Pending counts how many points are being buffered by the NATS client
	metricPendingNodePoint     *client.Metric
	metricPendingNodeEdgePoint *client.Metric

	chStop        chan struct{}
	chStopMetrics chan struct{}
	chWaitStart   chan struct{}
}

// Params are used to configure a store
type Params struct {
	File      string
	AuthToken string
	Server    string
	Nc        *nats.Conn
	// ID for the instance -- it is only used when initializing the store.
	// ID must be unique. If ID is not set, then a UUID is generated.
	ID string
}

// NewStore creates a new NATS client for handling SIOT requests
func NewStore(p Params) (*Store, error) {
	db, err := NewSqliteDb(p.File, p.ID)
	if err != nil {
		return nil, fmt.Errorf("error opening db: %v", err)
	}

	// we don't have node ID yet, but need to init here so we can start
	// collecting data

	authorizer, err := api.NewKey(db.meta.JWTKey)
	if err != nil {
		return nil, fmt.Errorf("error creating authorizer: %v", err)
	}

	log.Println("store connecting to nats server:", p.Server)
	return &Store{
		params:        p,
		nc:            p.Nc,
		db:            db,
		authorizer:    authorizer,
		subscriptions: make(map[string]*nats.Subscription),
		chStop:        make(chan struct{}),
		chStopMetrics: make(chan struct{}),
		chWaitStart:   make(chan struct{}),
		metricCycleNodePoint: client.NewMetric(p.Nc, "",
			data.PointTypeMetricNatsCycleNodePoint, reportMetricsPeriod),
		metricCycleNodeEdgePoint: client.NewMetric(p.Nc, "",
			data.PointTypeMetricNatsCycleNodeEdgePoint, reportMetricsPeriod),
		metricCycleNode: client.NewMetric(p.Nc, "",
			data.PointTypeMetricNatsCycleNode, reportMetricsPeriod),
		metricCycleNodeChildren: client.NewMetric(p.Nc, "",
			data.PointTypeMetricNatsCycleNodeChildren, reportMetricsPeriod),
	}, nil
}

// GetAuthorizer returns a type that can be used in JWT Auth mechanisms
func (st *Store) GetAuthorizer() api.Authorizer {
	return st.authorizer
}

// Run connects to NATS server and set up handlers for things we are interested in
func (st *Store) Run() error {
	nc := st.params.Nc
	var err error
	st.subscriptions["nodePoints"], err = nc.Subscribe("p.*", st.handleNodePoints)
	if err != nil {
		return fmt.Errorf("subscribe node points error: %w", err)
	}

	st.subscriptions["edgePoints"], err = nc.Subscribe("p.*.*", st.handleEdgePoints)
	if err != nil {
		return fmt.Errorf("subscribe edge points error: %w", err)
	}

	if st.subscriptions["nodes"], err = nc.Subscribe("nodes.*.*", st.handleNodesRequest); err != nil {
		return fmt.Errorf("subscribe node error: %w", err)
	}

	/*
		if st.subscriptions["notifications"], err = nc.Subscribe("node.*.not", st.handleNotification); err != nil {
			return fmt.Errorf("Subscribe notification error: %w", err)
		}

		if st.subscriptions["messages"], err = nc.Subscribe("node.*.msg", st.handleMessage); err != nil {
			return fmt.Errorf("Subscribe message error: %w", err)
		}
	*/

	if st.subscriptions["auth.user"], err = nc.Subscribe("auth.user", st.handleAuthUser); err != nil {
		return fmt.Errorf("subscribe auth error: %w", err)
	}

	if st.subscriptions["auth.getNatsURI"], err = nc.Subscribe("auth.getNatsURI", st.handleAuthGetNatsURI); err != nil {
		return fmt.Errorf("subscribe auth error: %w", err)
	}

	if st.subscriptions["admin.storeVerify"], err = nc.Subscribe("admin.storeVerify", st.handleStoreVerify); err != nil {
		return fmt.Errorf("subscribe dbVerify error: %w", err)
	}

	if st.subscriptions["admin.storeMaint"], err = nc.Subscribe("admin.storeMaint", st.handleStoreMaint); err != nil {
		return fmt.Errorf("subscribe dbMaint error: %w", err)
	}

done:
	for {
		select {
		case <-st.chWaitStart:
			// don't need to do anything as simply reading this
			// channel will unblock the caller
		case <-st.chStop:
			log.Println("Store stopped")
			break done
		}
	}

	// clean up
	for k := range st.subscriptions {
		err := st.subscriptions[k].Unsubscribe()
		if err != nil {
			log.Printf("Error unsubscribing from %v: %v\n", k, err)
		}
	}

	st.db.Close()

	return nil
}

// Stop the store
func (st *Store) Stop(_ error) {
	close(st.chStop)
}

// WaitStart waits for store to start
func (st *Store) WaitStart(ctx context.Context) error {
	waitDone := make(chan struct{})

	go func() {
		// the following will block until the main store select
		// loop starts
		st.chWaitStart <- struct{}{}
		close(waitDone)
	}()

	select {
	case <-ctx.Done():
		return errors.New("Store wait timeout or canceled")
	case <-waitDone:
		// all is well
		return nil
	}
}

// Reset the store by permanently wiping all data
func (st *Store) Reset() error {
	return st.db.reset()
}

// StartMetrics for various handling operations. Metrics are sent to the node ID given
// FIXME, this can probably move to the node package for device nodes
func (st *Store) StartMetrics(nodeID string) error {
	st.metricCycleNodePoint.SetNodeID(nodeID)
	st.metricCycleNodeEdgePoint.SetNodeID(nodeID)
	st.metricCycleNode.SetNodeID(nodeID)
	st.metricCycleNodeChildren.SetNodeID(nodeID)

	st.metricPendingNodePoint = client.NewMetric(st.nc, nodeID,
		data.PointTypeMetricNatsPendingNodePoint, reportMetricsPeriod)
	st.metricPendingNodeEdgePoint = client.NewMetric(st.nc, nodeID,
		data.PointTypeMetricNatsPendingNodeEdgePoint, reportMetricsPeriod)

	t := time.NewTimer(time.Millisecond)

	for {
		select {
		case <-st.chStopMetrics:
			return errors.New("Store stopping metrics")

		case <-t.C:
			pendingNodePoints, _, err := st.subscriptions["nodePoints"].Pending()
			if err != nil {
				log.Println("Error getting pendingNodePoints:", err)
			}

			err = st.metricPendingNodePoint.AddSample(float64(pendingNodePoints))
			if err != nil {
				log.Println("Error handling metric:", err)
			}

			pendingEdgePoints, _, err := st.subscriptions["edgePoints"].Pending()
			if err != nil {
				log.Println("Error getting pendingEdgePoints:", err)
			}

			err = st.metricPendingNodeEdgePoint.AddSample(float64(pendingEdgePoints))
			if err != nil {
				log.Println("Error handling metric:", err)
			}
			t.Reset(time.Second * 10)
		}
	}
}

// StopMetrics ...
func (st *Store) StopMetrics(_ error) {
	close(st.chStopMetrics)
}

func (st *Store) handleNodePoints(msg *nats.Msg) {
	start := time.Now()
	defer func() {
		t := time.Since(start).Milliseconds()
		err := st.metricCycleNodePoint.AddSample(float64(t))
		if err != nil {
			log.Println("Error stopping metrics:", err)
		}
	}()

	nodeID, points, err := client.DecodeNodePointsMsg(msg)

	if err != nil {
		fmt.Printf("Error decoding nats message: %v: %v", msg.Subject, err)
		st.reply(msg.Reply, errors.New("error decoding node points subject"))
		return
	}

	// write points to database
	err = st.db.nodePoints(nodeID, points)

	if err != nil {
		// TODO track error stats
		log.Printf("Error writing nodeID (%v) to Db: %v", nodeID, err)
		log.Println("msg subject:", msg.Subject)
		st.reply(msg.Reply, err)
		return
	}

	// process point in upstream nodes
	err = st.processPointsUpstream(nodeID, nodeID, points)
	if err != nil {
		// TODO track error stats
		log.Println("Error processing point in upstream nodes:", err)
	}

	st.reply(msg.Reply, nil)
}

func (st *Store) handleEdgePoints(msg *nats.Msg) {
	start := time.Now()
	defer func() {
		t := time.Since(start).Milliseconds()
		err := st.metricCycleNodeEdgePoint.AddSample(float64(t))
		if err != nil {
			log.Println("handle edge point error:", err)
		}
	}()

	nodeID, parentID, points, err := client.DecodeEdgePointsMsg(msg)

	if err != nil {
		fmt.Printf("Error decoding nats message: %v: %v", msg.Subject, err)
		st.reply(msg.Reply, errors.New("error decoding edge points subject"))
		return
	}

	// write points to database. Its important that we write to the DB
	// before sending points upstream, or clients may do a rescan and not
	// see the node is deleted.
	err = st.db.edgePoints(nodeID, parentID, points)

	if err != nil {
		// TODO track error stats
		log.Printf("Error writing edge points (%v:%v) to Db: %v", nodeID, parentID, err)
		st.reply(msg.Reply, err)
	}

	// process point in upstream nodes. We need to do this before writing
	// to DB, otherwise the point will not be sent upstream
	err = st.processEdgePointsUpstream(nodeID, nodeID, parentID, points)
	if err != nil {
		// TODO track error stats
		log.Println("Error processing point in upstream nodes:", err)
	}

	st.reply(msg.Reply, nil)
}

func (st *Store) handleNodesRequest(msg *nats.Msg) {
	start := time.Now()
	defer func() {
		t := time.Since(start).Milliseconds()
		err := st.metricCycleNode.AddSample(float64(t))
		if err != nil {
			log.Println("handleNodesRequest error:", err)
		}
	}()

	resp := &pb.NodesRequest{}
	var err error
	var parent string
	var nodeID string
	var includeDel bool
	var nodeType string
	var nodes data.Nodes

	chunks := strings.Split(msg.Subject, ".")
	if len(chunks) < 3 {
		resp.Error = fmt.Sprintf("Error in message subject: %v", msg.Subject)
		goto handleNodeDone
	}

	parent = chunks[1]
	nodeID = chunks[2]

	if len(msg.Data) > 0 {
		pts, err := data.PbDecodePoints(msg.Data)
		if err != nil {
			resp.Error = fmt.Sprintf("Error decoding points %v", err)
			goto handleNodeDone
		}

		for _, p := range pts {
			switch p.Type {
			case data.PointTypeTombstone:
				includeDel = data.FloatToBool(p.Value)
			case data.PointTypeNodeType:
				nodeType = p.Text
			}
		}
	}

	nodes, err = st.db.getNodes(nil, parent, nodeID, nodeType, includeDel)

	if err != nil {
		if err != data.ErrDocumentNotFound {
			resp.Error = fmt.Sprintf("NATS handler: Error getting node %v from db: %v\n", nodeID, err)
		} else {
			resp.Error = data.ErrDocumentNotFound.Error()
		}
	}

handleNodeDone:
	resp.Nodes, err = nodes.ToPbNodes()
	if err != nil {
		resp.Error = fmt.Sprintf("Error pb encoding node: %v\n", err)
	}

	data, err := proto.Marshal(resp)
	if err != nil {
		log.Println("marshal error:", err)
		return
	}

	err = st.nc.Publish(msg.Reply, data)
	if err != nil {
		log.Println("NATS: Error publishing response to node request:", err)
	}
}

// TODO, maybe someday we should return error node instead of no data
func (st *Store) handleAuthUser(msg *nats.Msg) {
	var points data.Points
	var err error
	resp := &pb.NodesRequest{}

	returnNothing := func() {
		err = st.nc.Publish(msg.Reply, nil)
		if err != nil {
			log.Println("NATS: Error publishing response to auth.user")
		}
	}

	if len(msg.Data) <= 0 {
		log.Println("No data in auth.user")
		returnNothing()
		return
	}

	points, err = data.PbDecodePoints(msg.Data)
	if err != nil {
		log.Println("Error decoding auth.user params:", err)
		returnNothing()
		return
	}

	emailP, ok := points.Find(data.PointTypeEmail, "")
	if !ok {
		log.Println("Error, auth.user no email point")
		returnNothing()
		return
	}

	passP, ok := points.Find(data.PointTypePass, "")
	if !ok {
		log.Println("Error, auth.user no password point")
		returnNothing()
		return
	}

	nodes, err := st.db.userCheck(emailP.Text, passP.Text)

	if err != nil || len(nodes) <= 0 {
		log.Println("Error, invalid user")
		returnNothing()
		return
	}

	user, err := data.NodeToUser(nodes[0].ToNode())

	token, err := st.authorizer.NewToken(user.ID)
	if err != nil {
		log.Println("Error creating token")
		returnNothing()
		return
	}

	nodes = append(nodes, data.NodeEdge{
		Type: data.NodeTypeJWT,
		Points: data.Points{
			{
				Type: data.PointTypeToken,
				Text: token,
				Key:  "0",
			},
		},
	})

	resp.Nodes, err = nodes.ToPbNodes()
	if err != nil {
		resp.Error = fmt.Sprintf("Error pb encoding node: %v\n", err)
	}

	data, err := proto.Marshal(resp)

	err = st.nc.Publish(msg.Reply, data)
	if err != nil {
		log.Println("NATS: Error publishing response to node request:", err)
	}
}

func (st *Store) handleAuthGetNatsURI(msg *nats.Msg) {
	points := data.Points{
		{Type: data.PointTypeURI, Text: st.params.Server},
		{Type: data.PointTypeToken, Text: st.params.AuthToken},
	}

	data, err := points.ToPb()

	if err != nil {
		data = []byte(err.Error())
	}

	err = st.nc.Publish(msg.Reply, data)
	if err != nil {
		log.Println("NATS: Error publishing response to gets NATS URI request:", err)
	}
}

func (st *Store) handleStoreVerify(msg *nats.Msg) {
	var ret string
	hashErr := st.db.verifyNodeHashes(false)
	if hashErr != nil {
		ret = hashErr.Error()
	}

	err := st.nc.Publish(msg.Reply, []byte(ret))
	if err != nil {
		log.Println("NATS: Error publishing response to node request:", err)
	}
}

func (st *Store) handleStoreMaint(msg *nats.Msg) {
	var ret string
	hashErr := st.db.verifyNodeHashes(true)
	if hashErr != nil {
		ret = hashErr.Error()
	}

	err := st.nc.Publish(msg.Reply, []byte(ret))
	if err != nil {
		log.Println("NATS: Error publishing response to node request:", err)
	}
}

// used for messages that want an ACK
func (st *Store) reply(subject string, err error) {
	if subject == "" {
		// node is not expecting a reply
		return
	}

	reply := ""

	if err != nil {
		reply = err.Error()
	}

	e := st.nc.Publish(subject, []byte(reply))
	if e != nil {
		log.Println("Error ack reply:", e)
	}
}

func (st *Store) processPointsUpstream(upNodeID, nodeID string, points data.Points) error {
	// at this point, the point update has already been written to the DB
	sub := fmt.Sprintf("up.%v.%v", upNodeID, nodeID)

	err := client.SendPoints(st.nc, sub, points, false)

	if err != nil {
		return err
	}

	if upNodeID == "none" {
		// we are at the top, stop
		return nil
	}

	ups, err := st.db.up(upNodeID, false)
	if err != nil {
		return err
	}

	for _, up := range ups {
		err = st.processPointsUpstream(up, nodeID, points)
		if err != nil {
			log.Println("Rules -- error processing upstream node:", err)
		}
	}

	/* FIXME needs to be move to client

	if currentNodeID == nodeID {
		// check if device node that it has not been orphaned
		node, err := st.db.node(nodeID)
		if err != nil {
			log.Println("Error getting node:", err)
		}

		if node.Type == data.NodeTypeDevice {
			hasUpstream := false
			for _, e := range edges {
				if !e.IsTombstone() {
					hasUpstream = true
				}
			}

			if !hasUpstream {
				fmt.Println("STORE: orphaned node: ", node)
				if len(edges) < 1 {
					// create upstream edge
					err := client.SendEdgePoint(st.nc, nodeID, "none", data.Point{
						Type:  data.PointTypeTombstone,
						Value: 0,
					}, false)
					if err != nil {
						log.Println("Error sending edge point:", err)
					}
				} else {
					// undelete existing edge
					e := edges[0]
					err := client.SendEdgePoint(st.nc, e.Down, e.Up, data.Point{
						Type:  data.PointTypeTombstone,
						Value: 0,
					}, false)
					if err != nil {
						log.Println("Error sending edge point:", err)
					}
				}
			}
		}
	}
	*/

	return nil
}

func (st *Store) processEdgePointsUpstream(upNodeID, nodeID, parentID string, points data.Points) error {
	sub := fmt.Sprintf("up.%v.%v.%v", upNodeID, nodeID, parentID)

	err := client.SendPoints(st.nc, sub, points, false)

	if err != nil {
		return err
	}

	if upNodeID == "none" {
		// we are at the top, stop
		return nil
	}

	ups, err := st.db.up(upNodeID, true)
	if err != nil {
		return err
	}

	for _, up := range ups {
		err = st.processEdgePointsUpstream(up, nodeID, parentID, points)
		if err != nil {
			log.Println("Rules -- error processing upstream node:", err)
		}
	}

	return nil
}
