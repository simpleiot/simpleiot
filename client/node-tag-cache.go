package client

import (
	"fmt"
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
	Tags map[tagEntry]string
}
type nodeCache struct {
	// TagPointTypes is a slice of point types that are added as Influx tags
	TagPointTypes []string
	// Cache is a map of cache entries
	Cache map[string]nodeCacheEntry
	// Lock is the cache mutex
	Lock *sync.RWMutex
}

// newNodeCache returns an initialized nodeCache
func newNodeCache(tagPointTypes []string) nodeCache {
	tagPointTypes = slices.Clone(tagPointTypes)
	slices.Sort(tagPointTypes)
	return nodeCache{
		// We sort the slice, so we can use BinarySearch
		TagPointTypes: tagPointTypes,
		Cache:         make(map[string]nodeCacheEntry),
		Lock:          new(sync.RWMutex),
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
// does not exist for the node, the node is retrieved, and the cache is
// subsequently updated.
func (c nodeCache) Update(nc *nats.Conn, pts NewPoints) error {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	entry, found := c.Cache[pts.ID]
	if !found {
		// We need to fetch the node and populate the cache
		ne, err := GetNodes(nc, "all", pts.ID, "", false)
		if err != nil {
			return err
		}
		if len(ne) <= 0 {
			return fmt.Errorf("Tag Cache, node of ID %v not found in DB", pts.ID)
		}
		entry.Type = ne[0].Type
		entry.Tags = make(map[tagEntry]string)
		for _, p := range ne[0].Points {
			if p.Tombstone%2 == 1 {
				continue
			}
			if p.Type == data.PointTypeDescription {
				entry.Description = p.Text
			}
			if _, found := slices.BinarySearch(c.TagPointTypes, p.Type); found {
				key := tagEntry{Type: p.Type, Key: p.Key}
				entry.Tags[key] = p.Text
			}
		}
	}

	// Update the entry from the specified points
	for _, p := range pts.Points {
		if p.Type == data.PointTypeDescription {
			if p.Tombstone%2 == 0 {
				entry.Description = p.Text
			} else {
				entry.Description = ""
			}
		}
		if _, found := slices.BinarySearch(c.TagPointTypes, p.Type); found {
			key := tagEntry{Type: p.Type, Key: p.Key}
			if p.Tombstone%2 == 0 && p.Text != "" {
				entry.Tags[key] = p.Text
			} else {
				delete(entry.Tags, key)
			}
		}
	}
	c.Cache[pts.ID] = entry

	return nil
}

// Clear deletes all cache entries
func (c *nodeCache) Clear() {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	c.Cache = make(map[string]nodeCacheEntry)
}
