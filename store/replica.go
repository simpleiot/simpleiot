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
}

// replicaManager tracks running replica stream consumers.
type replicaManager struct {
	db      *DbJetStream
	mu      sync.Mutex
	running map[string]jetstream.ConsumeContext
	// unknown records replica streams whose boundary is not a node this
	// instance knows, so each is reported once. The permission set is what
	// keeps a device inside its own boundary; this is the store noticing
	// when something got past it, or when a device was removed from the
	// tree and its stream is still here.
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
// Deliveries merge into the caches from the start of the stream (the
// merge is idempotent against the startup pre-population); broadcasts
// are gated until the backlog drains.
func (rm *replicaManager) consumeReplica(name, origin string) (jetstream.ConsumeContext, error) {
	ctx := context.Background()

	s, err := rm.db.js.Stream(ctx, name)
	if err != nil {
		return nil, err
	}

	c, err := s.OrderedConsumer(ctx, jetstream.OrderedConsumerConfig{})
	if err != nil {
		return nil, err
	}

	// pending holds the last raw payload per storage subject that
	// changed a tip during catch-up; flushed as one broadcast per
	// subject when the backlog drains
	pending := make(map[string][]byte)
	caughtUp := false

	return c.Consume(func(msg jetstream.Msg) {
		changed := rm.db.mergeReplicaMsg(msg.Subject(), msg.Data(), origin)

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
func (db *DbJetStream) mergeReplicaMsg(subject string, payload []byte, origin string) bool {
	tok := strings.Split(subject, ".")

	pts, err := data.DecodePoints(payload)
	if err != nil {
		log.Printf("STORE: error decoding replica msg %v: %v", subject, err)
		return false
	}

	switch {
	case len(tok) == 7 && tok[4] == "p":
		nodeID := tok[3]
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
		nodeType := ""
		for _, p := range pts {
			if p.Type == data.PointTypeNodeType {
				nodeType = p.Txt()
			}
		}
		return db.edgeCache.MergeEdgePoints(parentID, childID, nodeType, origin, pts)
	}

	return false
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
