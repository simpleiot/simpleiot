package store

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/simpleiot/simpleiot/data"
)

// Meta contains metadata about the store instance, persisted in the
// META KV bucket.
type Meta struct {
	RootID string `json:"rootID"`
	JWTKey []byte `json:"jwtKey"`
}

// JsConfig holds JetStream store tunables.
type JsConfig struct {
	// MaxMsgsPerSubject bounds per-subject history in the streams this
	// instance creates; 0 or less means unlimited. Retention is per
	// subject, so subject tips (current state) are always preserved,
	// including rarely-updated config points that time- or size-based
	// policies could silently drop. Changing the value applies to each
	// existing stream the first time it is ensured after restart, and
	// JetStream trims existing subjects to the new limit.
	MaxMsgsPerSubject int64
}

// DbJetStream implements the store backend using NATS JetStream with
// boundary-origin streams (ADR-7): each (boundary, origin instance)
// pair gets one stream holding both node points and edge points for
// the nodes the boundary owns. Only the origin instance appends to a
// stream. Current state is the merge of subject tips across the
// streams for a boundary, held in the in-memory edge and point caches,
// which are fully populated at startup and are the read path.
type DbJetStream struct {
	js        jetstream.JetStream
	nc        *nats.Conn
	metaKV    jetstream.KeyValue
	meta      Meta
	cfg       JsConfig
	edgeCache *EdgeCache

	pointMu    sync.RWMutex
	pointCache map[string]data.Points // nodeID -> current point tips
	// pointOrigin tracks which instance wrote each tip
	// (nodeID -> "type|key" -> origin instance ID)
	pointOrigin map[string]map[string]string

	// streams caches handles for origin streams this instance has
	// ensured, so the ensure path hits the JetStream API once per
	// stream per process
	streamMu sync.Mutex
	streams  map[string]jetstream.Stream
}

// streamName returns the stream name for a (boundary, origin) pair.
// Stream names cannot contain dots, so names use dashes.
func streamName(boundaryID, originID string) string {
	return "inst-" + boundaryID + "-" + originID
}

// streamCaptureSubject returns the subject space a boundary-origin
// stream captures. Subject spaces never overlap between streams
// because both routing tokens are in every subject.
func streamCaptureSubject(boundaryID, originID string) string {
	return fmt.Sprintf("inst.%v.%v.>", boundaryID, originID)
}

// streamBoundaryOrigin extracts the boundary and origin IDs from a
// stream's capture subject. ok is false for streams that are not
// boundary-origin streams (for example KV backing streams).
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

// nodePointSubject returns the storage subject for a node point.
func nodePointSubject(boundaryID, originID, nodeID, typ, key string) string {
	if key == "" {
		key = "0"
	}
	return fmt.Sprintf("inst.%v.%v.%v.p.%v.%v", boundaryID, originID, nodeID, typ, key)
}

// edgePointSubject returns the storage subject for edge points. Edges
// are stored with the parent node's boundary.
func edgePointSubject(boundaryID, originID, parentID, childID string) string {
	return fmt.Sprintf("inst.%v.%v.%v.ep.%v", boundaryID, originID, parentID, childID)
}

// NewJetStreamDb creates a new JetStream-backed store.
func NewJetStreamDb(nc *nats.Conn, rootID string, cfg JsConfig) (*DbJetStream, error) {
	// Wait for NATS connection to be established (server may not be up yet)
	timeout := time.After(10 * time.Second)
	for !nc.IsConnected() {
		select {
		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for NATS connection")
		case <-time.After(50 * time.Millisecond):
		}
	}

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
		js:          js,
		nc:          nc,
		metaKV:      metaKV,
		cfg:         cfg,
		edgeCache:   NewEdgeCache(),
		pointCache:  make(map[string]data.Points),
		pointOrigin: make(map[string]map[string]string),
		streams:     make(map[string]jetstream.Stream),
	}

	// Load meta from KV
	err = db.loadMeta()
	if err != nil {
		return nil, fmt.Errorf("error loading meta: %v", err)
	}

	// Mandatory cache pre-population: read every boundary-origin
	// stream's subject tips into the edge and point caches before
	// serving anything, so the caches are the read path and a partial
	// entry can never be trusted
	err = db.loadAllStreams()
	if err != nil {
		return nil, fmt.Errorf("error loading streams: %v", err)
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

// ensureOriginStream creates (or updates) this instance's origin stream
// for a boundary. Local writes only ever go through origin streams;
// Stage 3 adds a separate path for replica streams sourced from remote
// origins, which are never published to directly.
func (db *DbJetStream) ensureOriginStream(boundaryID string) (jetstream.Stream, error) {
	return db.ensureOriginStreamFor(boundaryID, db.meta.RootID)
}

func (db *DbJetStream) ensureOriginStreamFor(boundaryID, originID string) (jetstream.Stream, error) {
	name := streamName(boundaryID, originID)

	db.streamMu.Lock()
	s, ok := db.streams[name]
	db.streamMu.Unlock()
	if ok {
		return s, nil
	}

	ctx := context.Background()
	s, err := db.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:              name,
		Subjects:          []string{streamCaptureSubject(boundaryID, originID)},
		MaxMsgsPerSubject: db.maxMsgsForStream(name),
	})
	if err != nil {
		return nil, fmt.Errorf("error creating stream %v: %v", name, err)
	}

	db.streamMu.Lock()
	db.streams[name] = s
	db.streamMu.Unlock()

	return s, nil
}

// maxMsgsForStream resolves the per-subject retention limit for a
// stream by name. Every stream currently gets the instance default;
// Stage 3 adds per-boundary and per-replica overrides here (a hub
// keeps long history on replicas while a device keeps a short local
// buffer).
func (db *DbJetStream) maxMsgsForStream(_ string) int64 {
	if db.cfg.MaxMsgsPerSubject > 0 {
		return db.cfg.MaxMsgsPerSubject
	}
	return 0
}

// mergePointTip merges a single point into the point cache, applying
// the ADR-7 tip merge rule. It returns true if the point became the
// current tip.
func (db *DbJetStream) mergePointTip(nodeID string, pIn data.Point, origin string) bool {
	if pIn.Key == "" {
		pIn.Key = "0"
	}
	k := pIn.Type + "|" + pIn.Key

	db.pointMu.Lock()
	defer db.pointMu.Unlock()

	origins := db.pointOrigin[nodeID]
	if origins == nil {
		origins = make(map[string]string)
		db.pointOrigin[nodeID] = origins
	}

	pts := db.pointCache[nodeID]
	for i, p := range pts {
		if p.Type == pIn.Type && p.Key == pIn.Key {
			if !tipWins(p.Time, origins[k], pIn.Time, origin) {
				return false
			}
			// copy-on-write so slices handed out by getNodes are not
			// mutated underneath readers
			npts := append(data.Points{}, pts...)
			npts[i] = pIn
			db.pointCache[nodeID] = npts
			origins[k] = origin
			return true
		}
	}

	db.pointCache[nodeID] = append(pts, pIn)
	origins[k] = origin
	return true
}

// pointIsTip reports whether the point would become the current tip if
// written now.
func (db *DbJetStream) pointIsTip(nodeID string, pIn data.Point, origin string) bool {
	if pIn.Key == "" {
		pIn.Key = "0"
	}
	k := pIn.Type + "|" + pIn.Key

	db.pointMu.RLock()
	defer db.pointMu.RUnlock()

	for _, p := range db.pointCache[nodeID] {
		if p.Type == pIn.Type && p.Key == pIn.Key {
			return tipWins(p.Time, db.pointOrigin[nodeID][k], pIn.Time, origin)
		}
	}
	return true
}

// nodePoints writes node points to this instance's origin stream for
// the node's owning boundary and updates the point cache.
func (db *DbJetStream) nodePoints(id string, points data.Points) error {
	points.Collapse()

	origin := db.meta.RootID
	boundary := db.edgeCache.OwningBoundary(id, origin)

	// backstop for the startup pre-population: on a cache miss, load
	// the node's current points from JetStream before adding new ones,
	// so the cache can never be seeded from a partial write
	err := db.ensureNodePointsCached(boundary, id)
	if err != nil {
		return err
	}

	_, err = db.ensureOriginStream(boundary)
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

		if !db.pointIsTip(id, pIn, origin) {
			log.Println("Ignoring node point due to timestamps:", id, pIn)
			continue
		}

		subject := nodePointSubject(boundary, origin, id, pIn.Type, pIn.Key)
		pts := data.Points{pIn}
		_, err = db.js.Publish(ctx, subject, pts.Encode())
		if err != nil {
			return fmt.Errorf("error publishing point to %v: %v", subject, err)
		}

		db.mergePointTip(id, pIn, origin)
	}

	return nil
}

// edgePoints writes edge points to JetStream and updates the edge
// cache. Edges are stored with the parent node's boundary.
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

	origin := db.meta.RootID
	var boundary string
	if parentID == "root" {
		// an instance root edge always lives in the root node's own
		// boundary-origin stream, so a fresh instance starts with the
		// single stream inst-<rootID>-<rootID>
		boundary = nodeID
		origin = nodeID
	} else {
		boundary = db.edgeCache.OwningBoundary(parentID, origin)
	}

	// capture the child's owning boundary before this edge lands so a
	// cross-boundary move can be detected afterward
	oldChildBoundary := db.edgeCache.OwningBoundary(nodeID, db.meta.RootID)

	s, err := db.ensureOriginStreamFor(boundary, origin)
	if err != nil {
		return err
	}

	ctx := context.Background()
	subject := edgePointSubject(boundary, origin, parentID, nodeID)

	// Extract nodeType from points (kept in persisted edge points
	// so it can be recovered on restart)
	var nodeType string
	var edgePoints data.Points
	for _, p := range points {
		if p.Type == data.PointTypeNodeType {
			nodeType = p.Txt()
		}
		edgePoints = append(edgePoints, p)
	}

	// Load existing edge points from this instance's own stream tip
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

		// an edge holds one point per type and key, so a repeat in the
		// incoming set replaces what was written for it rather than
		// being stored beside it
		dup := -1

		for i, pW := range writePoints {
			if pW.Type == pIn.Type && pW.Key == pIn.Key {
				dup = i
				break
			}
		}

		if dup >= 0 {
			if !writePoints[dup].Time.After(pIn.Time) {
				writePoints[dup] = pIn
			}

			continue
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

	// a new edge must carry a nodeType point; check before publishing so
	// the stream and edge cache cannot diverge
	entry, ok := db.edgeCache.Get(parentID, nodeID)
	if !ok && nodeType == "" {
		return fmt.Errorf("node type must be sent with new edges")
	}

	// a new edge or a tombstone transition can change the child's
	// owning boundary (a move lands as new-edge-then-tombstone, an
	// undelete restores a path); note it now, act after the cache
	// reflects the write
	wasTombstone := ok && entry.IsTombstone()
	newTombstone, _ := writePoints.ValueBool(data.PointTypeTombstone, "")
	boundaryCheck := !ok || wasTombstone != newTombstone

	// Publish merged edge points
	encoded := writePoints.Encode()
	_, err = db.js.Publish(ctx, subject, encoded)
	if err != nil {
		return fmt.Errorf("error publishing edge points to %v: %v", subject, err)
	}

	if !ok && parentID == "root" && nodeID != db.meta.RootID {
		log.Println("inserting new root node, update root in meta")
		_, err = db.metaKV.Put(ctx, "rootID", []byte(nodeID))
		if err != nil {
			return fmt.Errorf("error updating root id in meta: %v", err)
		}
		db.meta.RootID = nodeID
	}

	db.edgeCache.MergeEdgePoints(parentID, nodeID, nodeType, origin, writePoints)

	// a fully deleted node keeps its subjects where they are; an
	// undelete migrates them if ownership moved in the meantime
	if boundaryCheck && len(db.edgeCache.UpIDs(nodeID, false)) > 0 {
		newChildBoundary := db.edgeCache.OwningBoundary(nodeID, db.meta.RootID)
		if newChildBoundary != oldChildBoundary {
			err := db.migrateBoundary(nodeID, make(map[string]bool))
			if err != nil {
				log.Printf("STORE: error migrating %v to boundary %v: %v",
					nodeID, newChildBoundary, err)
			}
		}
	}

	return nil
}

// migrateBoundary handles a node whose owning boundary changed:
// republish its current subject tips into the new boundary's origin
// stream preserving original point timestamps, purge its subjects from
// every other local-origin stream, and walk its owned descendants doing
// the same. Descendants that are boundaries themselves own their
// subtree and do not move; tombstoned paths are left in place (an
// undelete migrates them when it restores ownership).
func (db *DbJetStream) migrateBoundary(nodeID string, visited map[string]bool) error {
	if visited[nodeID] {
		return nil
	}
	visited[nodeID] = true

	origin := db.meta.RootID
	boundary := db.edgeCache.OwningBoundary(nodeID, origin)

	db.pointMu.RLock()
	pts := append(data.Points{}, db.pointCache[nodeID]...)
	db.pointMu.RUnlock()

	children := db.edgeCache.Children(nodeID)

	// a freshly created node has nothing to move
	if len(pts) > 0 || len(children) > 0 {
		_, err := db.ensureOriginStream(boundary)
		if err != nil {
			return err
		}

		ctx := context.Background()

		// republish before purging so there is never a window with no
		// stored copy
		for _, p := range pts {
			subject := nodePointSubject(boundary, origin, nodeID, p.Type, p.Key)
			one := data.Points{p}
			_, err := db.js.Publish(ctx, subject, one.Encode())
			if err != nil {
				return fmt.Errorf("error republishing %v: %v", subject, err)
			}
		}

		// child edges belong to this node's boundary, tombstoned or not
		for _, e := range children {
			subject := edgePointSubject(boundary, origin, nodeID, e.Down)
			_, err := db.js.Publish(ctx, subject, e.Points.Encode())
			if err != nil {
				return fmt.Errorf("error republishing %v: %v", subject, err)
			}
		}

		err = db.purgeNodeSubjectsExcept(nodeID, boundary)
		if err != nil {
			return err
		}
	}

	for _, e := range children {
		if e.IsTombstone() {
			continue
		}
		if db.edgeCache.IsBoundary(e.Down, origin) {
			continue
		}
		err := db.migrateBoundary(e.Down, visited)
		if err != nil {
			return err
		}
	}

	return nil
}

// purgeNodeSubjectsExcept removes a node's subjects (its points and its
// child edges) from every local-origin stream other than the boundary
// that now owns it. Also used by permanent removal paths.
func (db *DbJetStream) purgeNodeSubjectsExcept(nodeID, keepBoundary string) error {
	ctx := context.Background()
	self := db.meta.RootID

	lister := db.js.ListStreams(ctx, jetstream.WithStreamListSubject("inst.>"))
	for si := range lister.Info() {
		b, o, ok := streamBoundaryOrigin(si.Config)
		if !ok || o != self || b == keepBoundary {
			continue
		}

		s, err := db.js.Stream(ctx, si.Config.Name)
		if err != nil {
			log.Printf("error getting stream %v: %v", si.Config.Name, err)
			continue
		}

		filter := fmt.Sprintf("inst.%v.%v.%v.>", b, o, nodeID)
		err = s.Purge(ctx, jetstream.WithPurgeSubject(filter))
		if err != nil {
			return fmt.Errorf("error purging %v: %v", filter, err)
		}
	}

	return lister.Err()
}

// loadAllStreams populates the edge and point caches from the subject
// tips of every boundary-origin stream.
func (db *DbJetStream) loadAllStreams() error {
	ctx := context.Background()

	lister := db.js.ListStreams(ctx, jetstream.WithStreamListSubject("inst.>"))
	for si := range lister.Info() {
		err := db.loadStream(si.Config)
		if err != nil {
			log.Printf("STORE: error loading stream %v: %v", si.Config.Name, err)
		}
	}

	return lister.Err()
}

// loadStream merges one stream's subject tips into the edge and point
// caches. It is safe to call concurrently with live writes and more
// than once per stream (the tip merge is idempotent); Stage 3 uses this
// same path when a replica stream appears at runtime.
func (db *DbJetStream) loadStream(cfg jetstream.StreamConfig) error {
	boundary, origin, ok := streamBoundaryOrigin(cfg)
	if !ok {
		// not a boundary-origin stream (e.g. a KV backing stream)
		return nil
	}

	ctx := context.Background()
	s, err := db.js.Stream(ctx, cfg.Name)
	if err != nil {
		return err
	}

	db.loadEdgeSubjects(s, origin, fmt.Sprintf("inst.%v.%v.*.ep.>", boundary, origin))
	db.loadPointSubjects(s, origin, fmt.Sprintf("inst.%v.%v.*.p.>", boundary, origin))

	return nil
}

// loadPointSubjects merges the tips of all node point subjects matching
// filter into the point cache.
func (db *DbJetStream) loadPointSubjects(s jetstream.Stream, origin, filter string) {
	ctx := context.Background()

	info, err := s.Info(ctx, jetstream.WithSubjectFilter(filter))
	if err != nil {
		log.Printf("error getting stream info for %v: %v", filter, err)
		return
	}

	for subject := range info.State.Subjects {
		// inst.<boundary>.<origin>.<nodeID>.p.<type>.<key>
		tok := strings.Split(subject, ".")
		if len(tok) != 7 || tok[4] != "p" {
			continue
		}

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

		for _, p := range pts {
			if p.Type == "" {
				p.Type = tok[5]
			}
			if p.Key == "" {
				p.Key = tok[6]
			}
			db.mergePointTip(tok[3], p, origin)
		}
	}
}

// loadEdgeSubjects merges the tips of all edge point subjects matching
// filter into the edge cache.
func (db *DbJetStream) loadEdgeSubjects(s jetstream.Stream, origin, filter string) {
	ctx := context.Background()

	info, err := s.Info(ctx, jetstream.WithSubjectFilter(filter))
	if err != nil {
		log.Printf("error getting stream info for %v: %v", filter, err)
		return
	}

	for subject := range info.State.Subjects {
		// inst.<boundary>.<origin>.<parentID>.ep.<childID>
		tok := strings.Split(subject, ".")
		if len(tok) != 6 || tok[4] != "ep" {
			continue
		}
		parentID, childID := tok[3], tok[5]

		msg, err := s.GetLastMsgForSubject(ctx, subject)
		if err != nil {
			log.Printf("error getting edge tip for %v: %v", subject, err)
			continue
		}

		pts, err := data.DecodePoints(msg.Data)
		if err != nil {
			log.Printf("error decoding edge points from %v: %v", subject, err)
			continue
		}

		nodeType := ""
		for _, p := range pts {
			if p.Type == data.PointTypeNodeType {
				nodeType = p.Txt()
				break
			}
		}

		db.edgeCache.MergeEdgePoints(parentID, childID, nodeType, origin, pts)
	}
}

// loadNodePoints merges one node's current points, read from all
// streams of its owning boundary, into the point cache.
func (db *DbJetStream) loadNodePoints(boundary, nodeID string) error {
	ctx := context.Background()

	lister := db.js.ListStreams(ctx,
		jetstream.WithStreamListSubject(fmt.Sprintf("inst.%v.>", boundary)))

	for si := range lister.Info() {
		b, o, ok := streamBoundaryOrigin(si.Config)
		if !ok {
			continue
		}

		s, err := db.js.Stream(ctx, si.Config.Name)
		if err != nil {
			log.Printf("error getting stream %v: %v", si.Config.Name, err)
			continue
		}

		db.loadPointSubjects(s, o, fmt.Sprintf("inst.%v.%v.%v.p.>", b, o, nodeID))
	}

	return lister.Err()
}

// ensureNodePointsCached makes sure the point cache holds an entry for
// the node, loading current points from JetStream on a miss. After
// this returns, an empty entry means the node has no points.
func (db *DbJetStream) ensureNodePointsCached(boundary, id string) error {
	db.pointMu.RLock()
	_, ok := db.pointCache[id]
	db.pointMu.RUnlock()
	if ok {
		return nil
	}

	err := db.loadNodePoints(boundary, id)
	if err != nil {
		return err
	}

	db.pointMu.Lock()
	if _, ok := db.pointCache[id]; !ok {
		db.pointCache[id] = data.Points{}
	}
	db.pointMu.Unlock()

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

		// The cache is the read path; a miss means the node has not
		// been seen since startup, so load it once as a backstop
		db.pointMu.RLock()
		points, ok := db.pointCache[edge.Down]
		db.pointMu.RUnlock()
		if !ok {
			boundary := db.edgeCache.OwningBoundary(edge.Down, db.meta.RootID)
			err := db.ensureNodePointsCached(boundary, edge.Down)
			if err != nil {
				log.Printf("error loading node points for %v: %v", edge.Down, err)
			}
			db.pointMu.RLock()
			points = db.pointCache[edge.Down]
			db.pointMu.RUnlock()
		}
		ne.Points = append(data.Points{}, points...)

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

	// set the instance identity before any writes so boundary and
	// origin routing resolve correctly during initialization
	db.meta.RootID = rootID

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

	// Delete all boundary-origin streams
	streamLister := db.js.ListStreams(ctx)
	for si := range streamLister.Info() {
		if strings.HasPrefix(si.Config.Name, "inst-") {
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

	// Reset caches
	db.edgeCache.Reset()
	db.pointMu.Lock()
	db.pointCache = make(map[string]data.Points)
	db.pointOrigin = make(map[string]map[string]string)
	db.pointMu.Unlock()
	db.streamMu.Lock()
	db.streams = make(map[string]jetstream.Stream)
	db.streamMu.Unlock()

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
