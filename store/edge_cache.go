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

// Reset clears all entries from the cache.
func (ec *EdgeCache) Reset() {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	ec.byUp = make(map[string][]EdgeEntry)
	ec.byDown = make(map[string][]EdgeEntry)
}
