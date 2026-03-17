package store

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/simpleiot/simpleiot/data"
)

// DbJetStream implements the store backend using NATS JetStream.
// Each node gets its own stream (node-<nodeID>) containing both
// node points and edge points. Current state is the tip of each
// subject; full history is retained for time-series use.
type DbJetStream struct {
	js        jetstream.JetStream
	nc        *nats.Conn
	metaKV    jetstream.KeyValue
	meta      Meta
	edgeCache *EdgeCache
}

// streamName converts a node ID to a JetStream stream name.
// Stream names cannot contain dots, so we use dashes.
func streamName(nodeID string) string {
	return "node-" + nodeID
}

// nodePointSubject returns the JetStream subject for a node point.
func nodePointSubject(nodeID, typ, key string) string {
	if key == "" {
		key = "0"
	}
	return fmt.Sprintf("node.%v.p.%v.%v", nodeID, typ, key)
}

// edgePointSubject returns the JetStream subject for edge points.
func edgePointSubject(parentID, childID string) string {
	return fmt.Sprintf("node.%v.ep.%v", parentID, childID)
}

// NewJetStreamDb creates a new JetStream-backed store.
func NewJetStreamDb(nc *nats.Conn, rootID string) (*DbJetStream, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("error creating JetStream context: %v", err)
	}

	ctx := context.Background()

	// Create or get META KV bucket
	metaKV, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket: "META",
	})
	if err != nil {
		return nil, fmt.Errorf("error creating META KV bucket: %v", err)
	}

	db := &DbJetStream{
		js:        js,
		nc:        nc,
		metaKV:    metaKV,
		edgeCache: NewEdgeCache(),
	}

	// Load meta from KV
	err = db.loadMeta()
	if err != nil {
		return nil, fmt.Errorf("error loading meta: %v", err)
	}

	// Load edge cache from existing streams
	err = db.loadEdgeCache()
	if err != nil {
		return nil, fmt.Errorf("error loading edge cache: %v", err)
	}

	if db.meta.RootID == "" {
		db.meta.RootID, err = db.initRoot(rootID)
		if err != nil {
			return nil, fmt.Errorf("error initializing root node: %v", err)
		}
	}

	if len(db.meta.JWTKey) == 0 {
		err = db.initJwtKey()
		if err != nil {
			return nil, fmt.Errorf("error initializing JWT key: %v", err)
		}
	}

	return db, nil
}

func (db *DbJetStream) loadMeta() error {
	ctx := context.Background()

	entry, err := db.metaKV.Get(ctx, "rootID")
	if err != nil && !errors.Is(err, jetstream.ErrKeyNotFound) {
		return err
	}
	if err == nil {
		db.meta.RootID = string(entry.Value())
	}

	entry, err = db.metaKV.Get(ctx, "jwtKey")
	if err != nil && !errors.Is(err, jetstream.ErrKeyNotFound) {
		return err
	}
	if err == nil {
		db.meta.JWTKey = entry.Value()
	}

	return nil
}

func (db *DbJetStream) initJwtKey() error {
	db.meta.JWTKey = make([]byte, 20)
	_, err := rand.Read(db.meta.JWTKey)
	if err != nil {
		return fmt.Errorf("error generating JWT key: %v", err)
	}

	ctx := context.Background()
	_, err = db.metaKV.Put(ctx, "jwtKey", db.meta.JWTKey)
	if err != nil {
		return fmt.Errorf("error storing JWT key: %v", err)
	}

	return nil
}

// ensureStream creates a per-node stream if it doesn't already exist.
func (db *DbJetStream) ensureStream(nodeID string) (jetstream.Stream, error) {
	ctx := context.Background()
	name := streamName(nodeID)

	s, err := db.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     name,
		Subjects: []string{fmt.Sprintf("node.%v.>", nodeID)},
	})
	if err != nil {
		return nil, fmt.Errorf("error creating stream %v: %v", name, err)
	}

	return s, nil
}

// getStream returns an existing stream for a node, or nil if it doesn't exist.
func (db *DbJetStream) getStream(nodeID string) (jetstream.Stream, error) {
	ctx := context.Background()
	s, err := db.js.Stream(ctx, streamName(nodeID))
	if err != nil {
		if errors.Is(err, jetstream.ErrStreamNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return s, nil
}

// nodePoints writes node points to JetStream, merging with existing data.
func (db *DbJetStream) nodePoints(id string, points data.Points) error {
	points.Collapse()

	s, err := db.ensureStream(id)
	if err != nil {
		return err
	}

	ctx := context.Background()

	for _, pIn := range points {
		if pIn.Time.IsZero() {
			pIn.Time = time.Now()
		}
		if pIn.Key == "" {
			pIn.Key = "0"
		}

		subject := nodePointSubject(id, pIn.Type, pIn.Key)

		// Check existing point
		existing, err := s.GetLastMsgForSubject(ctx, subject)
		if err != nil && !errors.Is(err, jetstream.ErrMsgNotFound) {
			return fmt.Errorf("error getting last msg for %v: %v", subject, err)
		}

		if existing != nil {
			// Decode existing point and compare timestamps
			existingPts, decErr := data.DecodePoints(existing.Data)
			if decErr == nil && len(existingPts) > 0 {
				if existingPts[0].Time.After(pIn.Time) {
					log.Println("Ignoring node point due to timestamps:", id, pIn)
					continue
				}
			}
		}

		// Encode and publish
		pts := data.Points{pIn}
		encoded := pts.Encode()
		_, err = db.js.Publish(ctx, subject, encoded)
		if err != nil {
			return fmt.Errorf("error publishing point to %v: %v", subject, err)
		}
	}

	return nil
}

// edgePoints writes edge points to JetStream and updates the edge cache.
func (db *DbJetStream) edgePoints(nodeID, parentID string, points data.Points) error {
	points.Collapse()

	if nodeID == parentID {
		return fmt.Errorf("error: edgePoints nodeID=parentID=%v", nodeID)
	}

	if nodeID == db.meta.RootID {
		for _, p := range points {
			if p.Type == data.PointTypeTombstone && p.Val() > 0 {
				return fmt.Errorf("error, can't delete root node")
			}
		}
	}

	if parentID == "" {
		parentID = "root"
	}

	// Ensure parent stream exists (edges stored under parent)
	s, err := db.ensureStream(parentID)
	if err != nil {
		return err
	}

	ctx := context.Background()
	subject := edgePointSubject(parentID, nodeID)

	// Extract nodeType from points (not stored as a regular edge point)
	var nodeType string
	var edgePoints data.Points
	for _, p := range points {
		if p.Type == data.PointTypeNodeType {
			nodeType = p.Txt()
			continue
		}
		edgePoints = append(edgePoints, p)
	}

	// Load existing edge points
	var dbPoints data.Points
	existing, err := s.GetLastMsgForSubject(ctx, subject)
	if err != nil && !errors.Is(err, jetstream.ErrMsgNotFound) {
		return fmt.Errorf("error getting last edge msg for %v: %v", subject, err)
	}
	if existing != nil {
		dbPoints, _ = data.DecodePoints(existing.Data)
	}

	// Merge: newer timestamps win
	var writePoints data.Points
	for _, pIn := range edgePoints {
		if pIn.Time.IsZero() {
			pIn.Time = time.Now()
		}
		if pIn.Key == "" {
			pIn.Key = "0"
		}

		found := false
		for _, pDb := range dbPoints {
			if pIn.Type == pDb.Type && pIn.Key == pDb.Key {
				found = true
				if pDb.Time.Before(pIn.Time) || pDb.Time.Equal(pIn.Time) {
					writePoints = append(writePoints, pIn)
				} else {
					log.Println("Ignoring edge point due to timestamps:", nodeID, pIn)
					writePoints = append(writePoints, pDb)
				}
				break
			}
		}
		if !found {
			writePoints = append(writePoints, pIn)
		}
	}

	// Keep existing points that weren't in the incoming set
	for _, pDb := range dbPoints {
		found := false
		for _, pIn := range edgePoints {
			if pIn.Type == pDb.Type && pIn.Key == pDb.Key {
				found = true
				break
			}
		}
		if !found {
			writePoints = append(writePoints, pDb)
		}
	}

	// Publish merged edge points
	encoded := writePoints.Encode()
	_, err = db.js.Publish(ctx, subject, encoded)
	if err != nil {
		return fmt.Errorf("error publishing edge points to %v: %v", subject, err)
	}

	// Update edge cache
	entry, ok := db.edgeCache.Get(parentID, nodeID)
	if !ok {
		if nodeType == "" {
			return fmt.Errorf("node type must be sent with new edges")
		}
		entry = EdgeEntry{
			Up:   parentID,
			Down: nodeID,
			Type: nodeType,
		}

		if parentID == "root" {
			log.Println("inserting new root node, update root in meta")
			_, err = db.metaKV.Put(ctx, "rootID", []byte(nodeID))
			if err != nil {
				return fmt.Errorf("error updating root id in meta: %v", err)
			}
			db.meta.RootID = nodeID
		}
	}
	if nodeType != "" {
		entry.Type = nodeType
	}
	entry.Points = writePoints
	db.edgeCache.Set(entry)

	return nil
}

// loadNodePoints loads all current points for a node from JetStream.
func (db *DbJetStream) loadNodePoints(nodeID string) (data.Points, error) {
	s, err := db.getStream(nodeID)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, nil
	}

	ctx := context.Background()

	// Get stream info with subject filter to find all point subjects
	filter := fmt.Sprintf("node.%v.p.>", nodeID)
	info, err := s.Info(ctx, jetstream.WithSubjectFilter(filter))
	if err != nil {
		return nil, fmt.Errorf("error getting stream info for %v: %v", nodeID, err)
	}

	var points data.Points
	for subject := range info.State.Subjects {
		msg, err := s.GetLastMsgForSubject(ctx, subject)
		if err != nil {
			log.Printf("error getting last msg for %v: %v", subject, err)
			continue
		}

		pts, err := data.DecodePoints(msg.Data)
		if err != nil {
			log.Printf("error decoding point from %v: %v", subject, err)
			continue
		}

		// Fill in type/key from subject if not set
		// Subject format: node.<nodeID>.p.<type>.<key>
		parts := strings.Split(subject, ".")
		if len(parts) >= 5 {
			for i := range pts {
				if pts[i].Type == "" {
					pts[i].Type = parts[3]
				}
				if pts[i].Key == "" {
					pts[i].Key = parts[4]
				}
			}
		}

		points = append(points, pts...)
	}

	return points, nil
}

// loadEdgePoints loads edge points for a specific edge from JetStream.
func (db *DbJetStream) loadEdgePoints(parentID, childID string) (data.Points, error) {
	s, err := db.getStream(parentID)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, nil
	}

	ctx := context.Background()
	subject := edgePointSubject(parentID, childID)
	msg, err := s.GetLastMsgForSubject(ctx, subject)
	if err != nil {
		if errors.Is(err, jetstream.ErrMsgNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return data.DecodePoints(msg.Data)
}

// loadEdgeCache populates the edge cache from all existing streams.
func (db *DbJetStream) loadEdgeCache() error {
	ctx := context.Background()

	// List all streams with the node- prefix
	streamLister := db.js.ListStreams(ctx)

	for si := range streamLister.Info() {
		name := si.Config.Name
		if !strings.HasPrefix(name, "node-") {
			continue
		}

		parentID := strings.TrimPrefix(name, "node-")

		// Find edge subjects in this stream
		filter := fmt.Sprintf("node.%v.ep.>", parentID)
		info, err := db.js.Stream(ctx, name)
		if err != nil {
			log.Printf("error getting stream %v: %v", name, err)
			continue
		}

		sInfo, err := info.Info(ctx, jetstream.WithSubjectFilter(filter))
		if err != nil {
			log.Printf("error getting stream info for %v: %v", name, err)
			continue
		}

		for subject := range sInfo.State.Subjects {
			// Subject format: node.<parentID>.ep.<childID>
			parts := strings.Split(subject, ".")
			if len(parts) < 4 {
				continue
			}
			childID := parts[3]

			msg, err := info.GetLastMsgForSubject(ctx, subject)
			if err != nil {
				log.Printf("error getting edge tip for %v: %v", subject, err)
				continue
			}

			pts, err := data.DecodePoints(msg.Data)
			if err != nil {
				log.Printf("error decoding edge points from %v: %v", subject, err)
				continue
			}

			// Determine node type from edge points or default
			nodeType := ""
			for _, p := range pts {
				if p.Type == data.PointTypeNodeType {
					nodeType = p.Txt()
					break
				}
			}

			db.edgeCache.Set(EdgeEntry{
				Up:     parentID,
				Down:   childID,
				Type:   nodeType,
				Points: pts,
			})
		}
	}

	return nil
}

// getNodes retrieves nodes based on parent/id/type filters.
// If parent is "all", all instances of node id are returned.
// If parent is set and id is "all", all children are returned.
// If parent is "root" and id is "all", the root node is returned.
func (db *DbJetStream) getNodes(_ any, parent, id, typ string, includeDel bool) ([]data.NodeEdge, error) {
	if parent == "" || parent == "none" {
		return nil, errors.New("parent must be set to valid ID, or all")
	}

	if id == "" {
		id = "all"
	}

	var edges []EdgeEntry

	switch {
	case parent == "root":
		edges = db.edgeCache.Children("root")
		if id != "all" {
			// Filter to specific root node
			var filtered []EdgeEntry
			for _, e := range edges {
				if e.Down == id {
					filtered = append(filtered, e)
				}
			}
			edges = filtered
		}
	case parent == "all" && id == "all":
		return nil, errors.New("invalid combination of parent and id")
	case parent == "all":
		edges = db.edgeCache.Parents(id)
	case id == "all":
		edges = db.edgeCache.Children(parent)
	default:
		e, ok := db.edgeCache.Get(parent, id)
		if ok {
			edges = []EdgeEntry{e}
		}
	}

	if typ != "" {
		var filtered []EdgeEntry
		for _, e := range edges {
			if e.Type == typ {
				filtered = append(filtered, e)
			}
		}
		edges = filtered
	}

	var ret []data.NodeEdge
	for _, edge := range edges {
		ne := data.NodeEdge{
			ID:         edge.Down,
			Parent:     edge.Up,
			Type:       edge.Type,
			EdgePoints: edge.Points,
		}

		if !includeDel {
			tombstone, _ := ne.IsTombstone()
			if tombstone {
				continue
			}
		}

		// Load node points
		points, err := db.loadNodePoints(edge.Down)
		if err != nil {
			log.Printf("error loading node points for %v: %v", edge.Down, err)
		}
		ne.Points = points

		ret = append(ret, ne)
	}

	return ret, nil
}

// up returns upstream node IDs for a given node.
func (db *DbJetStream) up(id string, includeDeleted bool) ([]string, error) {
	return db.edgeCache.UpIDs(id, includeDeleted), nil
}

// userCheck checks user authentication.
func (db *DbJetStream) userCheck(email, password string) (data.Nodes, error) {
	// Find all user-type edges
	userEdges := db.edgeCache.AllByType(data.NodeTypeUser)

	var users []data.NodeEdge

	for _, edge := range userEdges {
		ne, err := db.getNodes(nil, "all", edge.Down, "", false)
		if err != nil {
			log.Println("Error getting user node for id:", edge.Down)
			continue
		}
		if len(ne) < 1 {
			continue
		}

		n := ne[0].ToNode()
		u := n.ToUser()
		if u.Email == email && u.Pass == password {
			users = append(users, ne...)
		}
	}

	// Verify each user has a path to root
	var ret []data.NodeEdge
	for _, u := range users {
		if db.hasPathToRoot(u.ID) {
			ret = append(ret, u)
		}
	}

	return ret, nil
}

// hasPathToRoot checks if a node has an undeleted path to the root node.
func (db *DbJetStream) hasPathToRoot(id string) bool {
	parents := db.edgeCache.Parents(id)
	for _, e := range parents {
		if e.IsTombstone() {
			continue
		}
		if e.Up == "root" {
			return true
		}
		if db.hasPathToRoot(e.Up) {
			return true
		}
	}
	return false
}

func (db *DbJetStream) initRoot(rootID string) (string, error) {
	log.Println("STORE: Initialize root node and admin user")

	if rootID == "" {
		rootID = uuid.New().String()
	}

	// Create root node edge
	err := db.edgePoints(rootID, "root", data.Points{
		data.NewPointFloat(data.PointTypeTombstone, "", 0),
		data.NewPointString(data.PointTypeNodeType, "", data.NodeTypeDevice),
	})
	if err != nil {
		return "", fmt.Errorf("error sending root node edges: %w", err)
	}

	// Create admin user
	admin := data.User{
		ID:        uuid.New().String(),
		FirstName: "admin",
		LastName:  "user",
		Email:     "admin",
		Pass:      "admin",
	}

	points := admin.ToPoints()

	err = db.nodePoints(admin.ID, points)
	if err != nil {
		return "", fmt.Errorf("error setting default user: %v", err)
	}

	err = db.edgePoints(admin.ID, rootID, data.Points{
		data.NewPointFloat(data.PointTypeTombstone, "", 0),
		data.NewPointString(data.PointTypeNodeType, "", data.NodeTypeUser),
	})
	if err != nil {
		return "", err
	}

	ctx := context.Background()
	_, err = db.metaKV.Put(ctx, "rootID", []byte(rootID))
	if err != nil {
		return "", fmt.Errorf("error setting meta rootID: %v", err)
	}

	return rootID, nil
}

// reset wipes all data and re-initializes.
func (db *DbJetStream) reset() error {
	ctx := context.Background()

	// Delete all node- streams
	streamLister := db.js.ListStreams(ctx)
	for si := range streamLister.Info() {
		if strings.HasPrefix(si.Config.Name, "node-") {
			err := db.js.DeleteStream(ctx, si.Config.Name)
			if err != nil {
				return fmt.Errorf("error deleting stream %v: %v", si.Config.Name, err)
			}
		}
	}

	// Clear META KV
	err := db.metaKV.Purge(ctx, "rootID")
	if err != nil && !errors.Is(err, jetstream.ErrKeyNotFound) {
		return fmt.Errorf("error purging rootID: %v", err)
	}

	// Reset edge cache
	db.edgeCache.Reset()

	// Preserve root ID and re-initialize
	db.meta.RootID, err = db.initRoot(db.meta.RootID)
	if err != nil {
		return fmt.Errorf("error initializing root node: %v", err)
	}

	return nil
}

// Close is a no-op for JetStream (managed by the NATS server).
func (db *DbJetStream) Close() error {
	return nil
}

func (db *DbJetStream) rootNodeID() string {
	return db.meta.RootID
}
