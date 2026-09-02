package store

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/api"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
)

var reportMetricsPeriod = time.Minute

// pointErrorPeriod is how often a node is told about points that were
// rejected. A device with a bad point type or key normally sends it at its
// full rate, and repeating the same error on every point helps no one.
var pointErrorPeriod = time.Minute

// Store implements the SIOT NATS api
type Store struct {
	params        Params
	nc            *nats.Conn
	subscriptions map[string]*nats.Subscription
	db            *DbJetStream
	authorizer    api.Key

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

	// when each node was last told about a point it sent that could not be
	// accepted, so a device sending bad points at full rate is reported
	// once rather than continuously
	pointErrMu   sync.Mutex
	pointErrLast map[string]time.Time
}

// Params are used to configure a store
type Params struct {
	AuthToken string
	Server    string
	Nc        *nats.Conn
	// ID for the instance -- it is only used when initializing the store.
	// ID must be unique. If ID is not set, then a UUID is generated.
	ID string
	// JsConfig holds JetStream tunables (retention, etc.)
	JsConfig JsConfig
}

// NewStore creates a new NATS client for handling SIOT requests
func NewStore(p Params) (*Store, error) {
	db, err := NewJetStreamDb(p.Nc, p.ID, p.JsConfig)
	if err != nil {
		return nil, fmt.Errorf("error opening db: %v", err)
	}

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
		pointErrLast:  make(map[string]time.Time),
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

// UserFromToken returns the user a JWT was issued to and when it expires.
// The NATS authorizer uses it to authenticate browser connections.
func (st *Store) UserFromToken(token string) (string, time.Time, bool) {
	return st.authorizer.TokenClaims(token)
}

// UserAnchors lists the nodes a user sits directly under, which are the
// subtrees the user may see. Empty for a user that is not in the tree.
func (st *Store) UserAnchors(userID string) []string {
	return st.db.userAnchors(userID)
}

// Run connects to NATS server and set up handlers for things we are interested in
func (st *Store) Run() error {
	nc := st.params.Nc
	var err error
	st.subscriptions["nodePoints"], err = nc.Subscribe("p.>", st.handleNodePoints)
	if err != nil {
		return fmt.Errorf("subscribe node points error: %w", err)
	}

	st.subscriptions["edgePoints"], err = nc.Subscribe("ep.*.*", st.handleEdgePoints)
	if err != nil {
		return fmt.Errorf("subscribe edge points error: %w", err)
	}

	if st.subscriptions["nodes"], err = nc.Subscribe("nodes.*.*", st.handleNodesRequest); err != nil {
		return fmt.Errorf("subscribe node error: %w", err)
	}

	if st.subscriptions["auth.user"], err = nc.Subscribe("auth.user", st.handleAuthUser); err != nil {
		return fmt.Errorf("subscribe auth error: %w", err)
	}

	if st.subscriptions["auth.me"], err = nc.Subscribe("auth.me", st.handleAuthMe); err != nil {
		return fmt.Errorf("subscribe auth error: %w", err)
	}

	// the user namespace: requests from browser connections, which the
	// NATS server has limited to u.<anchor>.<user>.> for the anchors
	// that user sits under
	if st.subscriptions["user"], err = nc.Subscribe("u.*.*.>", st.handleUserRequest); err != nil {
		return fmt.Errorf("subscribe user namespace error: %w", err)
	}

	if st.subscriptions["admin.storeVerify"], err = nc.Subscribe("admin.storeVerify", st.handleStoreVerify); err != nil {
		return fmt.Errorf("subscribe dbVerify error: %w", err)
	}

	if st.subscriptions["admin.storeMaint"], err = nc.Subscribe("admin.storeMaint", st.handleStoreMaint); err != nil {
		return fmt.Errorf("subscribe dbMaint error: %w", err)
	}

	// consume replica streams (remote-origin data) into the caches and
	// re-broadcast changed tips locally (ADR-7 Stage 3)
	replicas := st.db.runReplicaManager()

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
	replicas.Stop()

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

// checkPoints drops points whose type or key cannot be represented in a NATS
// subject and returns the ones that can, along with an error describing what
// was dropped.
//
// Every point the store accepts is fanned out on a subject built from its type
// and key, and listeners read the node ID and parent ID from fixed positions in
// that subject. A type or key carrying a period adds a token and shifts
// everything after it, so the point reaches the wrong handler. Checking here,
// at the one place every point enters the system, keeps every subject the store
// publishes well formed.
func (st *Store) checkPoints(nodeID string, points []data.Point) ([]data.Point, error) {
	accepted := make([]data.Point, 0, len(points))
	var rejected []string

	for _, p := range points {
		if err := p.CheckSubjectTokens(); err != nil {
			rejected = append(rejected, err.Error())
			continue
		}

		accepted = append(accepted, p)
	}

	if len(rejected) == 0 {
		return points, nil
	}

	err := fmt.Errorf("node %v: %v", nodeID, strings.Join(rejected, "; "))

	log.Println("Store: rejected points:", err)
	st.reportPointError(nodeID, err)

	return accepted, err
}

// reportPointError sets an error point on a node so that a device sending a
// point the system cannot accept is visible in the UI, rather than only in the
// server log.
func (st *Store) reportPointError(nodeID string, err error) {
	st.pointErrMu.Lock()
	last, reported := st.pointErrLast[nodeID]
	if reported && time.Since(last) < pointErrorPeriod {
		st.pointErrMu.Unlock()
		return
	}
	st.pointErrLast[nodeID] = time.Now()
	st.pointErrMu.Unlock()

	// the error point carries no type or key of its own that could be
	// rejected, so this cannot loop
	e := client.SendNodePoint(st.nc, nodeID,
		data.NewPointString(data.PointTypeError, "", err.Error()), false)
	if e != nil {
		log.Println("Store: error reporting rejected point to node:", e)
	}
}

// hashPassPoints replaces the value of any pass point with its bcrypt hash.
// Pass points hold user passwords, which are stored only as hashes. Hashing
// here covers every local write path (UI, API, import, provisioning) in one
// place; a new node's points can arrive before its edge, so this cannot key
// off the node type. Already-hashed and empty values pass through unchanged,
// which keeps rehashes and remote-origin data written locally idempotent. A
// point whose value bcrypt cannot hash (over 72 bytes) is dropped and
// reported as a point error on the node.
func (st *Store) hashPassPoints(nodeID string, points data.Points) data.Points {
	accepted := points[:0]

	for _, p := range points {
		if p.Type == data.PointTypePass && p.Txt() != "" &&
			!data.PasswordIsHashed(p.Txt()) {
			hash, err := data.HashPassword(p.Txt())
			if err != nil {
				err = fmt.Errorf("node %v: error hashing pass point: %w",
					nodeID, err)
				log.Println("Store:", err)
				st.reportPointError(nodeID, err)
				continue
			}
			p.PutString(hash)
		}
		accepted = append(accepted, p)
	}

	return accepted
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

	points, errCheck := st.checkPoints(nodeID, points)
	if len(points) == 0 {
		st.reply(msg.Reply, errCheck)
		return
	}

	if origin := msg.Header.Get(client.OriginHeader); origin != "" &&
		origin != st.db.rootNodeID() {
		// points from another instance are merged for reads and fanned
		// out, but never persisted here: the replica stream is the
		// persistent copy (single-writer streams, ADR-7)
		st.db.mergeRemoteNodePoints(nodeID, points, origin)
	} else {
		points = st.hashPassPoints(nodeID, points)
		if len(points) == 0 {
			st.reply(msg.Reply, errCheck)
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
	}

	// process point in upstream nodes
	err = st.processPointsUpstream(nodeID, nodeID, points)
	if err != nil {
		// TODO track error stats
		log.Println("Error processing point in upstream nodes:", err)
	}

	// errCheck is nil unless part of this message was rejected
	st.reply(msg.Reply, errCheck)
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

	points, errCheck := st.checkPoints(nodeID, points)
	if len(points) == 0 {
		st.reply(msg.Reply, errCheck)
		return
	}

	if origin := msg.Header.Get(client.OriginHeader); origin != "" &&
		origin != st.db.rootNodeID() {
		// remote-origin edge points: merge and fan out only (see
		// handleNodePoints)
		st.db.mergeRemoteEdgePoints(nodeID, parentID, points, origin)
	} else {
		// write points to database. Its important that we write to the DB
		// before sending points upstream, or clients may do a rescan and not
		// see the node is deleted.
		err = st.db.edgePoints(nodeID, parentID, points)

		if err != nil {
			// TODO track error stats
			log.Printf("Error writing edge points (%v:%v) to Db: %v", nodeID, parentID, err)
			st.reply(msg.Reply, err)
			return
		}
	}

	// process point in upstream nodes
	err = st.processEdgePointsUpstream(nodeID, nodeID, parentID, points)
	if err != nil {
		// TODO track error stats
		log.Println("Error processing point in upstream nodes:", err)
	}

	// errCheck is nil unless part of this message was rejected
	st.reply(msg.Reply, errCheck)
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

	var respErr error
	var parent string
	var nodeID string
	var includeDel bool
	var nodeType string
	var depth int
	var nodes data.Nodes

	chunks := strings.Split(msg.Subject, ".")
	if len(chunks) < 3 {
		respErr = fmt.Errorf("error in message subject: %v", msg.Subject)
		goto handleNodeDone
	}

	parent = chunks[1]
	nodeID = chunks[2]

	if len(msg.Data) > 0 {
		pts, err := data.DecodePoints(msg.Data)
		if err != nil {
			respErr = fmt.Errorf("error decoding points %v", err)
			goto handleNodeDone
		}

		for _, p := range pts {
			switch p.Type {
			case data.PointTypeTombstone:
				includeDel = data.FloatToBool(p.Val())
			case data.PointTypeNodeType:
				nodeType = p.Txt()
			case data.PointTypeDepth:
				depth = int(p.Val())
			}
		}
	}

	nodes, respErr = st.db.getNodesDepth(parent, nodeID, nodeType, includeDel, depth)
	if respErr != nil && respErr != data.ErrDocumentNotFound {
		respErr = fmt.Errorf("NATS handler: Error getting node %v from db: %v", nodeID, respErr)
	}

handleNodeDone:
	err := st.nc.Publish(msg.Reply, data.EncodeNodes(nodes, respErr))
	if err != nil {
		log.Println("NATS: Error publishing response to node request:", err)
	}
}

// TODO, maybe someday we should return error node instead of no data
func (st *Store) handleAuthUser(msg *nats.Msg) {
	var points data.Points
	var err error

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

	points, err = data.DecodePoints(msg.Data)
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

	nodes, err := st.db.userCheck(emailP.Txt(), passP.Txt())

	if err != nil || len(nodes) <= 0 {
		log.Println("Error, invalid user")
		returnNothing()
		return
	}

	user, err := data.NodeToUser(nodes[0].ToNode())

	if user.Pass != "" && !data.PasswordIsHashed(user.Pass) {
		// successful login against a legacy plaintext password:
		// rewrite it as a hash through the normal point path
		hash, hashErr := data.HashPassword(user.Pass)
		if hashErr == nil {
			p := data.NewPointString(data.PointTypePass, "", hash)
			hashErr = client.SendNodePoint(st.nc, user.ID, p, false)
		}
		if hashErr != nil {
			log.Println("Error rehashing legacy password:", hashErr)
		}
	}

	token, err := st.authorizer.NewToken(user.ID)
	if err != nil {
		log.Println("Error creating token")
		returnNothing()
		return
	}

	nodes = append(nodes, data.NodeEdge{
		Type: data.NodeTypeJWT,
		Points: data.Points{
			data.NewPointString(data.PointTypeToken, "0", token),
		},
	})

	err = st.nc.Publish(msg.Reply, data.EncodeNodes(nodes, nil))
	if err != nil {
		log.Println("NATS: Error publishing response to auth.user request:", err)
	}
}

// handleAuthMe answers a browser asking who it is: the payload is the
// user's JWT, and the reply is the user's node at each place it sits in
// the tree, with the parent of each being an anchor the connection may
// reach. The password hash is left out.
func (st *Store) handleAuthMe(msg *nats.Msg) {
	reply := func(nodes data.Nodes, err error) {
		if e := st.nc.Publish(msg.Reply, data.EncodeNodes(nodes, err)); e != nil {
			log.Println("NATS: Error publishing response to auth.me:", e)
		}
	}

	userID, _, ok := st.authorizer.TokenClaims(string(msg.Data))
	if !ok {
		reply(nil, errors.New("invalid token"))
		return
	}

	edges, err := st.db.getNodes(nil, "all", userID, data.NodeTypeUser, false)
	if err != nil {
		reply(nil, err)
		return
	}

	var nodes data.Nodes
	for _, e := range edges {
		if e.Parent == "root" {
			continue
		}
		var pts data.Points
		for _, p := range e.Points {
			if p.Type != data.PointTypePass {
				pts = append(pts, p)
			}
		}
		e.Points = pts
		nodes = append(nodes, e)
	}

	if len(nodes) == 0 {
		reply(nil, errors.New("user is not in the tree"))
		return
	}

	reply(nodes, nil)
}

// handleUserRequest dispatches a request in the user namespace,
// u.<anchor>.<user>.<op>..., to the plain handler after checking that the
// target sits under the anchor. The NATS server has already proven that
// the connection may speak for this (anchor, user) pair, so the store only
// has to prove the target is in scope. Points are stamped with the user as
// their origin, and no header from the browser is passed on.
//
//	u.G.U.nodes.<parent>.<id>   the node (or the parent, when id is all) is under G
//	u.G.U.p.<id>.<type>.<key>   the node is under G
//	u.G.U.ep.<id>.<parent>      the parent is under G
func (st *Store) handleUserRequest(msg *nats.Msg) {
	tok := strings.Split(msg.Subject, ".")
	if len(tok) < 5 {
		st.reply(msg.Reply, fmt.Errorf("invalid subject: %v", msg.Subject))
		return
	}

	anchor, userID, op, rest := tok[1], tok[2], tok[3], tok[4:]

	refuse := func() {
		log.Printf("Store: refusing %v, target is not under %v", msg.Subject, anchor)
		st.reply(msg.Reply, errors.New("not in scope"))
	}

	switch op {
	case "nodes":
		if len(rest) != 2 {
			st.reply(msg.Reply, fmt.Errorf("invalid subject: %v", msg.Subject))
			return
		}
		parent, id := rest[0], rest[1]
		target := id
		if id == "all" {
			target = parent
		}
		if !st.db.isUnder(target, anchor) {
			// the reply is a node frame, so the error travels in one
			log.Printf("Store: refusing %v, target is not under %v", msg.Subject, anchor)
			if err := st.nc.Publish(msg.Reply, data.EncodeNodes(nil, errors.New("not in scope"))); err != nil {
				log.Println("NATS: Error publishing response to node request:", err)
			}
			return
		}
		st.handleNodesRequest(&nats.Msg{
			Subject: "nodes." + parent + "." + id,
			Reply:   msg.Reply,
			Data:    msg.Data,
		})

	case "p":
		if len(rest) != 3 {
			st.reply(msg.Reply, fmt.Errorf("invalid subject: %v", msg.Subject))
			return
		}
		if !st.db.isUnder(rest[0], anchor) {
			refuse()
			return
		}
		payload, err := stampOrigin(msg.Data, userID)
		if err != nil {
			st.reply(msg.Reply, err)
			return
		}
		st.handleNodePoints(&nats.Msg{
			Subject: "p." + strings.Join(rest, "."),
			Reply:   msg.Reply,
			Data:    payload,
		})

	case "ep":
		if len(rest) != 2 {
			st.reply(msg.Reply, fmt.Errorf("invalid subject: %v", msg.Subject))
			return
		}
		if !st.db.isUnder(rest[1], anchor) {
			refuse()
			return
		}
		payload, err := stampOrigin(msg.Data, userID)
		if err != nil {
			st.reply(msg.Reply, err)
			return
		}
		st.handleEdgePoints(&nats.Msg{
			Subject: "ep." + strings.Join(rest, "."),
			Reply:   msg.Reply,
			Data:    payload,
		})

	default:
		st.reply(msg.Reply, fmt.Errorf("unknown operation: %v", op))
	}
}

// stampOrigin re-encodes a points payload with every origin set to the
// user, so a point written from a browser records who wrote it whatever
// the browser said. A point with no time is given the current time, as
// the Go client does before sending.
func stampOrigin(payload []byte, userID string) ([]byte, error) {
	pts, err := data.DecodePoints(payload)
	if err != nil {
		return nil, fmt.Errorf("error decoding points: %w", err)
	}
	now := time.Now()
	for i := range pts {
		pts[i].Origin = userID
		// a zero time does not survive the encoding, so anything at
		// or before the epoch is taken as unset
		if pts[i].Time.Unix() <= 0 {
			pts[i].Time = now
		}
	}
	return pts.Encode(), nil
}

func (st *Store) handleStoreVerify(msg *nats.Msg) {
	// Hash verification is no longer needed with JetStream store
	err := st.nc.Publish(msg.Reply, []byte(""))
	if err != nil {
		log.Println("NATS: Error publishing response to store verify:", err)
	}
}

func (st *Store) handleStoreMaint(msg *nats.Msg) {
	// Hash maintenance is no longer needed with JetStream store
	err := st.nc.Publish(msg.Reply, []byte(""))
	if err != nil {
		log.Println("NATS: Error publishing response to store maint:", err)
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

// processPointsUpstream fans a node point out to every node above it in the
// tree. SendPoints appends the point type and key, so listeners see
// up.<upID>.<nodeID>.<type>.<key>, one token shorter than the edge point
// subject below. Listeners tell the two apart by counting tokens, so neither
// subject may gain or lose a token, and a point type or key may not contain a
// period -- see checkPoints, which is what keeps that true.
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
					err := client.SendEdgePoint(st.nc, nodeID, "none",
						data.NewPointFloat(data.PointTypeTombstone, "", 0), false)
					if err != nil {
						log.Println("Error sending edge point:", err)
					}
				} else {
					// undelete existing edge
					e := edges[0]
					err := client.SendEdgePoint(st.nc, e.Down, e.Up,
						data.NewPointFloat(data.PointTypeTombstone, "", 0), false)
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

// processEdgePointsUpstream fans an edge point up the tree. The subject carries
// the parent ID as well, which is how listeners tell edge points from node
// points -- see processPointsUpstream.
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
