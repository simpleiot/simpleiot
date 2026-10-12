package store

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
)

// This file implements the store side of ADR-7 Stage 3 synchronization:
// consuming replica streams. A replica stream is a local copy of a
// remote instance's origin stream (inst_<boundary>_<origin> with origin
// != this instance), appended by the sync client's replicator (or by
// JetStream sourcing when configured) — never by local point writes.
//
// The store runs a consumer on every replica stream. Each delivery is
// merged into the edge and point caches (idempotent, ADR-7 tip rule)
// and, when it changes a tip, re-broadcast on the core NATS wire
// subjects tagged with the origin header so local clients (rules, UI,
// up.> listeners) react to remote changes. During catch-up after an
// offline gap, broadcasts are held back and only the final tip of each
// changed subject is sent once the backlog drains, so state clients see
// converged state rather than a replay of intermediate points.

// mergeRemoteNodePoints merges points written by another instance into
// the point cache without persisting them.
func (db *DbJetStream) mergeRemoteNodePoints(id string, points data.Points, origin string) {
	for _, p := range points {
		db.mergePointTip(id, p, origin)
	}
}

// mergeRemoteEdgePoints merges edge points written by another instance
// into the edge cache without persisting them. Edges with the virtual
// "root" parent are instance-local (they anchor the writing instance's
// own tree) and are never merged from a remote origin.
func (db *DbJetStream) mergeRemoteEdgePoints(nodeID, parentID string, points data.Points, origin string) {
	if parentID == "root" {
		return
	}
	nodeType := ""
	for _, p := range points {
		if p.Type == data.PointTypeNodeType {
			nodeType = p.Txt()
		}
	}
	db.edgeCache.MergeEdgePoints(parentID, nodeID, nodeType, origin, points)
	db.releaseReplica(nodeID)
}

// replicaManager tracks running replica stream consumers.
type replicaManager struct {
	db      *DbJetStream
	mu      sync.Mutex
	running map[string]jetstream.ConsumeContext
	// unknown records replica streams whose boundary is not a node this
	// instance knows, so each is reported once: a device was removed from
	// the tree and its stream is still here. What a stream may write is
	// checked per message (see replicaWait).
	unknown map[string]bool
	stop    chan struct{}
	done    chan struct{}
}

const replicaScanPeriod = 3 * time.Second

// runReplicaManager starts the replica watcher: it scans for replica
// streams and runs a consumer on each until stop is closed.
func (db *DbJetStream) runReplicaManager() *replicaManager {
	rm := &replicaManager{
		db:      db,
		unknown: make(map[string]bool),
		running: make(map[string]jetstream.ConsumeContext),
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}

	go func() {
		defer close(rm.done)
		ticker := time.NewTicker(replicaScanPeriod)
		defer ticker.Stop()

		rm.scan()
		for {
			select {
			case <-rm.stop:
				rm.mu.Lock()
				for _, cc := range rm.running {
					cc.Stop()
				}
				rm.running = make(map[string]jetstream.ConsumeContext)
				rm.mu.Unlock()
				return
			case <-ticker.C:
				rm.scan()
			}
		}
	}()

	return rm
}

// Stop shuts the watcher and all replica consumers down.
func (rm *replicaManager) Stop() {
	close(rm.stop)
	<-rm.done
}

func (rm *replicaManager) scan() {
	ctx := context.Background()
	self := rm.db.meta.RootID

	lister := rm.db.js.ListStreams(ctx, jetstream.WithStreamListSubject("inst.>"))
	for si := range lister.Info() {
		boundary, origin, ok := streamBoundaryOrigin(si.Config)
		if !ok || origin == self {
			continue
		}

		rm.checkBoundary(si.Config.Name, boundary)

		rm.mu.Lock()
		_, already := rm.running[si.Config.Name]
		rm.mu.Unlock()
		if already {
			continue
		}

		// the local store owns stream configuration: the sync pumps
		// create replica streams bare, and this instance's storage
		// policy is applied when the stream is discovered
		rm.applyPolicy(si.Config)

		cc, err := rm.consumeReplica(si.Config.Name, origin)
		if err != nil {
			log.Printf("STORE: error consuming replica %v: %v", si.Config.Name, err)
			continue
		}

		log.Printf("STORE: consuming replica stream %v", si.Config.Name)
		rm.mu.Lock()
		rm.running[si.Config.Name] = cc
		rm.mu.Unlock()
	}
	if err := lister.Err(); err != nil {
		log.Println("STORE: error listing replica streams:", err)
	}
}

// checkBoundary reports, once, a replica stream for a boundary that is not
// a node in this instance's tree. The stream is still consumed: a device
// that was deleted and restored must not lose what it sent in between, and
// the edge that makes a boundary known can arrive after its data.
func (rm *replicaManager) checkBoundary(name, boundary string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.unknown[name] || boundary == rm.db.meta.RootID {
		return
	}

	if len(rm.db.edgeCache.Parents(boundary)) > 0 {
		return
	}

	rm.unknown[name] = true
	log.Printf("STORE: replica stream %v is for boundary %v, which is not a node in this tree",
		name, boundary)
}

// applyPolicy brings a replica stream's per-subject retention and file
// store compression in line with this instance's policy, leaving the
// rest of its configuration as is. Both are settings about how this
// instance stores data on its own disk, so a replica follows the local
// policy rather than the originating instance's.
func (rm *replicaManager) applyPolicy(cfg jetstream.StreamConfig) {
	wantMsgs := rm.db.maxMsgsForStream(cfg.Name)
	wantCompression := rm.db.compressionForStream(cfg.Name)

	if cfg.MaxMsgsPerSubject == wantMsgs && cfg.Compression == wantCompression {
		return
	}

	cfg.MaxMsgsPerSubject = wantMsgs
	cfg.Compression = wantCompression

	_, err := rm.db.js.UpdateStream(context.Background(), cfg)
	if err != nil {
		log.Printf("STORE: error applying storage policy to %v: %v",
			cfg.Name, err)
	}
}

// consumeReplica starts an ordered consumer on one replica stream.
// Only the tip of each subject matters to the caches, so the consumer
// starts at the last message per subject rather than replaying the whole
// stream, which on a large replica took minutes of CPU at every start.
// Tips already merged by the startup pre-population merge again as no
// change; broadcasts are gated until the backlog drains.
func (rm *replicaManager) consumeReplica(name, origin string) (jetstream.ConsumeContext, error) {
	ctx := context.Background()

	s, err := rm.db.js.Stream(ctx, name)
	if err != nil {
		return nil, err
	}

	c, err := s.OrderedConsumer(ctx, jetstream.OrderedConsumerConfig{
		DeliverPolicy: jetstream.DeliverLastPerSubjectPolicy,
	})
	if err != nil {
		return nil, err
	}

	// pending holds the last raw payload per storage subject that
	// changed a tip during catch-up; flushed as one broadcast per
	// subject when the backlog drains
	pending := make(map[string][]byte)
	caughtUp := false

	return c.Consume(func(msg jetstream.Msg) {
		changed := rm.db.mergeReplicaMsg(msg.Subject(), msg.Data(), origin, true)

		backlog := uint64(0)
		if meta, err := msg.Metadata(); err == nil {
			backlog = meta.NumPending
		}

		switch {
		case caughtUp:
			if changed {
				rm.db.broadcastReplica(msg.Subject(), msg.Data(), origin)
			}
		case backlog > 0:
			if changed {
				pending[msg.Subject()] = msg.Data()
			}
		default:
			// backlog drained: flush held tips, then this message
			for subj, payload := range pending {
				rm.db.broadcastReplica(subj, payload, origin)
			}
			pending = nil
			if changed {
				rm.db.broadcastReplica(msg.Subject(), msg.Data(), origin)
			}
			caughtUp = true
		}
	})
}

// mergeReplicaMsg merges one replica stream message into the caches,
// returning true if it changed a subject tip. The storage subject is
// inst.<boundary>.<origin>.<nodeID>.p.<type>.<key> for node points or
// inst.<boundary>.<origin>.<parentID>.ep.<childID> for edge points.
//
// A message from a downstream instance is merged only when it stays
// inside the stream's boundary (see replicaWait); one that does not is
// held, and merged if a later edge places it there. live says the
// message came from a replica consumer, so a held message is broadcast
// when it is released.
func (db *DbJetStream) mergeReplicaMsg(subject string, payload []byte, origin string, live bool) bool {
	tok := strings.Split(subject, ".")

	pts, err := data.DecodePoints(payload)
	if err != nil {
		log.Printf("STORE: error decoding replica msg %v: %v", subject, err)
		return false
	}

	switch {
	case len(tok) == 7 && tok[4] == "p":
		nodeID := tok[3]
		if wait := db.replicaWait(tok[1], origin, nodeID, ""); wait != "" {
			db.holdReplica(wait, subject, payload, origin, live)
			return false
		}
		changed := false
		for _, p := range pts {
			if p.Type == "" {
				p.Type = tok[5]
			}
			if p.Key == "" {
				p.Key = tok[6]
			}
			if db.mergePointTip(nodeID, p, origin) {
				changed = true
			}
		}
		return changed
	case len(tok) == 6 && tok[4] == "ep":
		parentID, childID := tok[3], tok[5]
		if parentID == "root" {
			// the replica's origin instance anchors its own tree with
			// the virtual "root" parent; that edge is instance-local
			// and must not become a second root here
			return false
		}
		if wait := db.replicaWait(tok[1], origin, parentID, childID); wait != "" {
			db.holdReplica(wait, subject, payload, origin, live)
			return false
		}
		nodeType := ""
		for _, p := range pts {
			if p.Type == data.PointTypeNodeType {
				nodeType = p.Txt()
			}
		}
		changed := db.edgeCache.MergeEdgePoints(parentID, childID, nodeType, origin, pts)
		db.releaseReplica(childID)
		return changed
	}

	return false
}

// replicaWait decides whether a replica stream message may be merged. A
// downstream instance writes its own boundary, so a point is merged only
// when its node is in the stream's boundary, and an edge only when its
// parent is and its child is not a node of this instance's own tree.
// Without this, a device could write the upstream root, or add a user
// under it and sign in with the whole tree in scope.
//
// Streams this instance wrote, and streams for its own root boundary
// (written by its upstream), are trusted. For the rest, replicaWait
// returns "" when the message may be merged, or else the node whose place
// in the tree has to change first: the message is held until an edge
// reaching that node is merged. childID is "" for a node point.
func (db *DbJetStream) replicaWait(boundary, origin, nodeID, childID string) string {
	self := db.meta.RootID
	if origin == self || boundary == self {
		return ""
	}

	if !db.inBoundary(nodeID, boundary) {
		return nodeID
	}

	if childID == "" {
		return ""
	}

	if db.ownedHere(childID) && !db.inBoundary(childID, boundary) {
		return childID
	}

	if _, ok := db.edgeCache.Get(nodeID, childID); !ok && db.isUnder(nodeID, childID) {
		// same rule as edgePoints: a new edge must not close a loop
		// through the tree
		return childID
	}

	return ""
}

// ownedHere reports whether a node is part of this instance's own tree
// rather than a device's: some chain of its edges, deleted or not, reaches
// this instance's root without passing through a device boundary. A
// device may place a node it created, or one another instance mirrored
// into a device, but not one of these. The answer does not depend on
// the order edges arrive in, so the start-up load and the live consumer
// agree.
func (db *DbJetStream) ownedHere(id string) bool {
	self := db.meta.RootID
	visited := map[string]bool{id: true}
	frontier := []string{id}

	for len(frontier) > 0 {
		n := frontier[0]
		frontier = frontier[1:]

		for _, e := range db.edgeCache.Parents(n) {
			up := e.Up
			if up == "root" || up == self {
				return true
			}
			if visited[up] {
				continue
			}
			visited[up] = true
			if db.edgeCache.IsBoundary(up, self) {
				continue
			}
			frontier = append(frontier, up)
		}
	}

	return false
}

// inBoundary reports whether a node is in a boundary: the boundary node
// itself, a node under it, or a deleted node every one of whose edges has
// a parent in it, so a downstream instance can still write the nodes it
// deleted and restore them.
func (db *DbJetStream) inBoundary(id, boundary string) bool {
	if id == boundary || db.isUnder(id, boundary) {
		return true
	}

	parents := db.edgeCache.Parents(id)
	if len(parents) == 0 {
		return false
	}
	for _, e := range parents {
		if e.Up != boundary && !db.isUnder(e.Up, boundary) {
			return false
		}
	}
	return true
}

// maxHeldReplica caps the replica messages held waiting for an edge. A
// message past the cap is dropped; it stays in its stream and is
// considered again at the next start.
const maxHeldReplica = 100000

// heldReplica is a replica stream message waiting for an edge.
type heldReplica struct {
	payload []byte
	origin  string
	live    bool
}

// holdReplica keeps a message refused by replicaWait until an edge
// reaches the node it waits on. A message arriving before the edge that
// places its node is normal (a consumer delivers the last message of
// each subject in stream order, not tree order), so each refusal is
// logged once per node, and a message still held says the downstream
// instance wrote outside its boundary.
func (db *DbJetStream) holdReplica(wait, subject string, payload []byte, origin string, live bool) {
	db.heldMu.Lock()
	defer db.heldMu.Unlock()

	byNode := db.held[wait]
	if byNode == nil {
		if db.heldCount >= maxHeldReplica {
			log.Printf("STORE: dropping replica msg %v: %v held already", subject, db.heldCount)
			return
		}
		byNode = make(map[string]heldReplica)
		db.held[wait] = byNode
		log.Printf("STORE: holding replica msg %v until node %v is in its boundary", subject, wait)
	}

	if _, ok := byNode[subject]; !ok {
		db.heldCount++
	}
	byNode[subject] = heldReplica{payload: payload, origin: origin, live: live}
}

// releaseReplica merges the held messages waiting on a node or any node
// below it, after an edge that may have placed them in their boundary.
// Those that are still outside it are held again.
func (db *DbJetStream) releaseReplica(id string) {
	db.heldMu.Lock()
	empty := len(db.held) == 0
	db.heldMu.Unlock()
	if empty {
		return
	}

	visited := map[string]bool{id: true}
	frontier := []string{id}

	for len(frontier) > 0 {
		n := frontier[0]
		frontier = frontier[1:]

		db.heldMu.Lock()
		byNode := db.held[n]
		delete(db.held, n)
		db.heldCount -= len(byNode)
		db.heldMu.Unlock()

		for subject, h := range byNode {
			if db.mergeReplicaMsg(subject, h.payload, h.origin, h.live) && h.live {
				db.broadcastReplica(subject, h.payload, h.origin)
			}
		}

		for _, e := range db.edgeCache.Children(n) {
			if e.IsTombstone() || visited[e.Down] {
				continue
			}
			visited[e.Down] = true
			frontier = append(frontier, e.Down)
		}
	}
}

// broadcastReplica re-broadcasts a replica stream message on the core
// NATS wire subjects, tagged with the origin header. The store's own
// wire handlers then fan it out to up.> subjects; the origin tag keeps
// them from persisting it again.
func (db *DbJetStream) broadcastReplica(subject string, payload []byte, origin string) {
	tok := strings.Split(subject, ".")

	var wire string
	switch {
	case len(tok) == 7 && tok[4] == "p":
		// inst.<b>.<o>.<nodeID>.p.<type>.<key> -> p.<nodeID>.<type>.<key>
		wire = client.SubjectNodePoint(tok[3], tok[5], tok[6])
	case len(tok) == 6 && tok[4] == "ep":
		// inst.<b>.<o>.<parentID>.ep.<childID> -> ep.<childID>.<parentID>
		wire = client.SubjectEdgePoints(tok[5], tok[3])
	default:
		return
	}

	msg := nats.NewMsg(wire)
	msg.Data = payload
	msg.Header.Set(client.OriginHeader, origin)
	err := db.nc.PublishMsg(msg)
	if err != nil {
		log.Printf("STORE: error broadcasting replica msg %v: %v", wire, err)
	}
}
