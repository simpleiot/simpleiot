package client

import (
	"fmt"
	"log"
	"slices"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

type tagEntry struct {
	Type string // Point Type
	Key  string // Point Key
}
type nodeCacheEntry struct {
	// Type is the cached node type
	Type string
	// Description is the cached node description
	Description string
	// Tags is a map of tags attached to this node, derived from the list of
	// points with a Type matching one of the TagPointTypes. Keys are a
	// concatenation of the point Type and point Key. Values are the point Text.
	// Tags set on the node itself are merged with tags inherited from its
	// ancestors up to the Boundary node; the value nearest the node wins.
	Tags map[tagEntry]string
}
type nodeCache struct {
	// TagPointTypes is a slice of point types that are added as Influx tags
	TagPointTypes []string
	// Boundary is the node ID at which the ancestor walk stops; its tags
	// are included. This is the Db client's configured Parent node.
	Boundary string
	// Cache is a map of cache entries
	Cache map[string]nodeCacheEntry
	// Lock is the cache mutex
	Lock *sync.RWMutex
	// TieLogged records tag keys for which an equal-depth ancestor tie has
	// already been logged, so each ambiguity is reported once
	TieLogged map[tagEntry]bool
	// fetch retrieves every living instance of a node with its Parent
	// populated. It is a field so tests can substitute a constructed tree.
	fetch func(nc *nats.Conn, id string) ([]data.NodeEdge, error)
}

// newNodeCache returns an initialized nodeCache. boundary is the node ID at
// which ancestor tag resolution stops (inclusive), normally the Db client's
// Parent node.
func newNodeCache(tagPointTypes []string, boundary string) nodeCache {
	tagPointTypes = slices.Clone(tagPointTypes)
	slices.Sort(tagPointTypes)
	return nodeCache{
		// We sort the slice, so we can use BinarySearch
		TagPointTypes: tagPointTypes,
		Boundary:      boundary,
		Cache:         make(map[string]nodeCacheEntry),
		Lock:          new(sync.RWMutex),
		TieLogged:     make(map[tagEntry]bool),
		fetch: func(nc *nats.Conn, id string) ([]data.NodeEdge, error) {
			return GetNodes(nc, "all", id, "", false)
		},
	}
}

// CopyTags finds the specified node in the cache and copies the node ID
// (into key "node.id"), the node description (into key "node.description"),
// the node type (into key "node.type"), and tags from the node's "tag" points
// (into key "node.tag.*" where * is the name of each tag) to the specified
// `tags` map, returning true if the node was found in the cache. If the node is
// not present in the cache, false is returned and tags is unmodified.
func (c nodeCache) CopyTags(nodeID string, tags map[string]string) bool {
	c.Lock.RLock()
	defer c.Lock.RUnlock()

	entry, found := c.Cache[nodeID]
	if !found {
		return false
	}

	tags["node.id"] = nodeID
	tags["node.description"] = entry.Description
	tags["node.type"] = entry.Type
	for tagEntry, val := range entry.Tags {
		tags["node."+tagEntry.Type+"."+tagEntry.Key] = val
	}
	return true
}

// Update iterates through each Point and updates the cache. If a cache entry
// does not exist for the node, the node is retrieved along with the tag
// points of its ancestors up to the Boundary node, and the cache is
// subsequently updated.
func (c nodeCache) Update(nc *nats.Conn, pts NewPoints) error {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	entry, found := c.Cache[pts.ID]
	if !found {
		// We need to fetch the node and populate the cache
		ne, err := c.fetch(nc, pts.ID)
		if err != nil {
			return err
		}
		if len(ne) <= 0 {
			return fmt.Errorf("tag Cache, node of ID %v not found in DB", pts.ID)
		}
		entry.Type = ne[0].Type
		entry.Tags = make(map[tagEntry]string)
		for _, p := range ne[0].Points {
			if p.Tombstone%2 == 1 {
				continue
			}
			if p.Type == data.PointTypeDescription {
				entry.Description = p.Txt()
			}
			if _, found := slices.BinarySearch(c.TagPointTypes, p.Type); found {
				key := tagEntry{Type: p.Type, Key: p.Key}
				entry.Tags[key] = p.Txt()
			}
		}
		err = c.mergeAncestorTags(nc, pts.ID, ne, entry.Tags)
		if err != nil {
			return err
		}
	}

	// Update the entry from the specified points
	for _, p := range pts.Points {
		if p.Type == data.PointTypeDescription {
			if p.Tombstone%2 == 0 {
				entry.Description = p.Txt()
			} else {
				entry.Description = ""
			}
		}
		if _, found := slices.BinarySearch(c.TagPointTypes, p.Type); found {
			key := tagEntry{Type: p.Type, Key: p.Key}
			if p.Tombstone%2 == 0 && p.Txt() != "" {
				entry.Tags[key] = p.Txt()
			} else {
				delete(entry.Tags, key)
			}
		}
	}
	c.Cache[pts.ID] = entry

	return nil
}

// mergeAncestorTags walks the ancestors of the given node breadth-first and
// merges their tag points into tags. Entries already present (from the node
// itself or a nearer ancestor) are not overwritten, so the value nearest the
// node wins. Within a level, nodes are applied in ascending node ID order,
// and the first time two nodes at the same depth define the same key a
// message is logged. The walk stops after the level containing the Boundary
// node (its tags are included) or when no living parents remain. instances
// is the fetch result for the node itself, used to seed the first level.
// Must be called with the write lock held.
func (c nodeCache) mergeAncestorTags(nc *nats.Conn, id string,
	instances []data.NodeEdge, tags map[tagEntry]string) error {
	if len(c.TagPointTypes) == 0 || id == c.Boundary {
		return nil
	}

	// a node can be reached by more than one path in the DAG; nearest
	// depth wins, so each ancestor is visited only once
	visited := map[string]bool{id: true}
	level := parentIDs(instances, visited)

	for len(level) > 0 {
		slices.Sort(level)

		// definedBy tracks which node defined each key at this level, to
		// detect equal-depth ties
		definedBy := make(map[tagEntry]string)
		var next []string
		boundaryReached := false

		for _, nid := range level {
			ne, err := c.fetch(nc, nid)
			if err != nil {
				return err
			}
			if len(ne) == 0 {
				continue
			}
			for _, p := range ne[0].Points {
				if p.Tombstone%2 == 1 || p.Txt() == "" {
					continue
				}
				if _, found := slices.BinarySearch(c.TagPointTypes, p.Type); !found {
					continue
				}
				key := tagEntry{Type: p.Type, Key: p.Key}
				if first, tie := definedBy[key]; tie {
					if !c.TieLogged[key] {
						c.TieLogged[key] = true
						log.Printf("Db tag cache: nodes %v and %v define tag %v:%v at the same depth; using the value from %v",
							first, nid, key.Type, key.Key, first)
					}
					continue
				}
				definedBy[key] = nid
				if _, found := tags[key]; !found {
					tags[key] = p.Txt()
				}
			}
			if nid == c.Boundary {
				boundaryReached = true
				continue
			}
			next = append(next, parentIDs(ne, visited)...)
		}

		if boundaryReached {
			break
		}
		level = next
	}

	return nil
}

// parentIDs returns the IDs of the living parents of the given node
// instances, skipping the top-level "root" marker and any node already
// visited. Returned IDs are marked visited.
func parentIDs(instances []data.NodeEdge, visited map[string]bool) []string {
	var ids []string
	for _, n := range instances {
		if n.Parent == "root" || n.Parent == "" || visited[n.Parent] {
			continue
		}
		visited[n.Parent] = true
		ids = append(ids, n.Parent)
	}
	return ids
}

// hasTagPointType reports whether any of the points has a type in the
// configured tag point types. Used to decide when a cache clear is needed,
// since a tag edit on any node can change the inherited tags of every node
// beneath it.
func (c nodeCache) hasTagPointType(pts data.Points) bool {
	for _, p := range pts {
		if _, found := slices.BinarySearch(c.TagPointTypes, p.Type); found {
			return true
		}
	}
	return false
}

// Clear deletes all cache entries
func (c *nodeCache) Clear() {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	c.Cache = make(map[string]nodeCacheEntry)
}
