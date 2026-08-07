package store

import (
	"sync"

	"github.com/simpleiot/simpleiot/data"
)

// EdgeEntry represents an edge relationship with its metadata
type EdgeEntry struct {
	Up     string
	Down   string
	Type   string
	Points data.Points
	// origins tracks which instance wrote the current tip of each
	// point ("type|key" -> origin instance ID) so MergeEdgePoints can
	// apply the ADR-7 tie-break deterministically
	origins map[string]string
}

// IsTombstone returns true if the edge is marked as deleted
func (e *EdgeEntry) IsTombstone() bool {
	tombstone, _ := e.Points.ValueBool(data.PointTypeTombstone, "")
	return tombstone
}

// EdgeCache provides fast in-memory lookups for edge relationships.
// It is populated on startup by reading edge subject tips from each
// node's stream and kept current as edge points arrive.
type EdgeCache struct {
	mu     sync.RWMutex
	byUp   map[string][]EdgeEntry // parentID -> children
	byDown map[string][]EdgeEntry // childID -> parents
}

// NewEdgeCache creates a new empty EdgeCache
func NewEdgeCache() *EdgeCache {
	return &EdgeCache{
		byUp:   make(map[string][]EdgeEntry),
		byDown: make(map[string][]EdgeEntry),
	}
}

// Set adds or updates an edge entry. If an edge with the same
// Up+Down already exists, it is replaced.
func (ec *EdgeCache) Set(entry EdgeEntry) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	ec.setByUp(entry)
	ec.setByDown(entry)
}

func (ec *EdgeCache) setByUp(entry EdgeEntry) {
	entries := ec.byUp[entry.Up]
	for i, e := range entries {
		if e.Down == entry.Down {
			entries[i] = entry
			return
		}
	}
	ec.byUp[entry.Up] = append(entries, entry)
}

func (ec *EdgeCache) setByDown(entry EdgeEntry) {
	entries := ec.byDown[entry.Down]
	for i, e := range entries {
		if e.Up == entry.Up {
			entries[i] = entry
			return
		}
	}
	ec.byDown[entry.Down] = append(entries, entry)
}

// Children returns all child edges for a given parent node ID.
func (ec *EdgeCache) Children(parentID string) []EdgeEntry {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	result := make([]EdgeEntry, len(ec.byUp[parentID]))
	copy(result, ec.byUp[parentID])
	return result
}

// Parents returns all parent edges for a given child node ID.
func (ec *EdgeCache) Parents(childID string) []EdgeEntry {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	result := make([]EdgeEntry, len(ec.byDown[childID]))
	copy(result, ec.byDown[childID])
	return result
}

// Get returns a specific edge entry, if it exists.
func (ec *EdgeCache) Get(parentID, childID string) (EdgeEntry, bool) {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	for _, e := range ec.byUp[parentID] {
		if e.Down == childID {
			return e, true
		}
	}
	return EdgeEntry{}, false
}

// UpIDs returns the upstream node IDs for a given child node.
// If includeDeleted is false, tombstoned edges are filtered out.
func (ec *EdgeCache) UpIDs(childID string, includeDeleted bool) []string {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	var ups []string
	for _, e := range ec.byDown[childID] {
		if includeDeleted || !e.IsTombstone() {
			ups = append(ups, e.Up)
		}
	}
	return ups
}

// AllByType returns all edge entries with the given node type.
func (ec *EdgeCache) AllByType(nodeType string) []EdgeEntry {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	var result []EdgeEntry
	for _, entries := range ec.byUp {
		for _, e := range entries {
			if e.Type == nodeType {
				result = append(result, e)
			}
		}
	}
	return result
}

// MergeEdgePoints merges an edge point set — a stream subject tip, or
// points just written locally — into the cache, applying the ADR-7 tip
// merge rule per point. origin is the instance that wrote the points.
// typ may be "" when the writer did not include a nodeType point and
// the edge is already known. It reports whether any point became a new
// tip.
func (ec *EdgeCache) MergeEdgePoints(up, down, typ, origin string, pts data.Points) bool {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	var entry EdgeEntry
	found := false
	for _, e := range ec.byUp[up] {
		if e.Down == down {
			entry = e
			found = true
			break
		}
	}

	if !found {
		entry = EdgeEntry{Up: up, Down: down}
	}
	if entry.origins == nil {
		entry.origins = make(map[string]string)
	}
	if typ != "" {
		entry.Type = typ
	}

	// copy-on-write so slices handed out by earlier lookups are not
	// mutated underneath readers
	merged := append(data.Points{}, entry.Points...)
	changed := !found

	for _, pIn := range pts {
		if pIn.Key == "" {
			pIn.Key = "0"
		}
		k := pIn.Type + "|" + pIn.Key

		idx := -1
		for i, p := range merged {
			if p.Type == pIn.Type && p.Key == pIn.Key {
				idx = i
				break
			}
		}

		if idx >= 0 {
			if !tipWins(merged[idx].Time, entry.origins[k], pIn.Time, origin) {
				continue
			}
			merged[idx] = pIn
		} else {
			merged = append(merged, pIn)
		}
		entry.origins[k] = origin
		changed = true
	}

	entry.Points = merged
	ec.setByUp(entry)
	ec.setByDown(entry)

	return changed
}

// IsBoundary returns true if the node is a stream boundary: the
// instance root node, or a device-type node, which corresponds to a
// potentially synced remote instance (see ADR-7 boundary-origin
// streams). rootID is the local instance root node ID.
func (ec *EdgeCache) IsBoundary(id, rootID string) bool {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	return ec.isBoundary(id, rootID)
}

// isBoundary must be called with ec.mu held.
func (ec *EdgeCache) isBoundary(id, rootID string) bool {
	if id == rootID {
		return true
	}

	// a node's type is the same on every parent edge, so any entry,
	// tombstoned or not, answers the question
	for _, e := range ec.byDown[id] {
		if e.Type == data.NodeTypeDevice {
			return true
		}
	}

	return false
}

// OwningBoundary returns the boundary node that owns the given node:
// the nearest boundary reachable walking up undeleted edges. A boundary
// node is owned by itself. A node reachable from no boundary, or from
// more than one, is owned by the instance root boundary. The walk stops
// at the first boundary on each path, so nodes inside a nested boundary
// belong to the inner one. rootID is the local instance root node ID.
func (ec *EdgeCache) OwningBoundary(id, rootID string) string {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	if ec.isBoundary(id, rootID) {
		return id
	}

	boundaries := make(map[string]bool)
	visited := map[string]bool{id: true}

	var walk func(n string)
	walk = func(n string) {
		for _, e := range ec.byDown[n] {
			if e.IsTombstone() {
				continue
			}
			up := e.Up
			if up == "root" {
				// top of the tree; only the instance root has the
				// virtual "root" parent, and it is a boundary itself,
				// so normally this is unreachable — resolve to root
				boundaries[rootID] = true
				continue
			}
			if visited[up] {
				continue
			}
			visited[up] = true
			if ec.isBoundary(up, rootID) {
				boundaries[up] = true
				continue
			}
			walk(up)
		}
	}
	walk(id)

	if len(boundaries) == 1 {
		for b := range boundaries {
			return b
		}
	}

	return rootID
}

// Reset clears all entries from the cache.
func (ec *EdgeCache) Reset() {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	ec.byUp = make(map[string][]EdgeEntry)
	ec.byDown = make(map[string][]EdgeEntry)
}
