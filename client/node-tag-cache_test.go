package client

import (
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// The tests drive tag resolution against constructed trees by replacing the
// cache's fetch function, so no server is needed. Tree keys are node IDs;
// each value holds one NodeEdge per living parent edge, mirroring what
// GetNodes(nc, "all", ...) returns.

func testTagCache(tagTypes []string, boundary string,
	tree map[string][]data.NodeEdge) nodeCache {
	c := newNodeCache(tagTypes, boundary)
	c.fetch = func(_ *nats.Conn, id string) ([]data.NodeEdge, error) {
		return tree[id], nil
	}
	return c
}

func tagNode(id, typ, parent string, points ...data.Point) data.NodeEdge {
	return data.NodeEdge{ID: id, Type: typ, Parent: parent,
		Points: data.Points(points)}
}

func tagPt(key, value string) data.Point {
	return data.NewPointString(data.PointTypeTag, key, value)
}

// resolve updates the cache for id and returns the resolved tag map
func resolve(t *testing.T, c nodeCache, id string) map[string]string {
	t.Helper()
	err := c.Update(nil, NewPoints{ID: id})
	if err != nil {
		t.Fatal("Error updating cache:", err)
	}
	tags := make(map[string]string)
	if !c.CopyTags(id, tags) {
		t.Fatal("node not found in cache after update:", id)
	}
	return tags
}

func checkTag(t *testing.T, tags map[string]string, key, want string) {
	t.Helper()
	if tags[key] != want {
		t.Errorf("tag %v: got %q, want %q", key, tags[key], want)
	}
}

// TestTagCacheAncestorInheritance covers inheritance from a grandparent and
// the nearest-ancestor override, matching the example tree in the plan.
func TestTagCacheAncestorInheritance(t *testing.T) {
	tree := map[string][]data.NodeEdge{
		"dev-root": {tagNode("dev-root", "device", "root",
			tagPt("site", "plant-a"), tagPt("customer", "acme"))},
		"press-3": {tagNode("press-3", "group", "dev-root",
			tagPt("machine", "press-3"), tagPt("site", "plant-b"))},
		"temp-1": {tagNode("temp-1", "modbusIo", "press-3",
			tagPt("sensor", "inlet"),
			data.NewPointString(data.PointTypeDescription, "", "inlet temp"))},
	}

	c := testTagCache([]string{data.PointTypeTag}, "dev-root", tree)
	tags := resolve(t, c, "temp-1")

	checkTag(t, tags, "node.id", "temp-1")
	checkTag(t, tags, "node.type", "modbusIo")
	checkTag(t, tags, "node.description", "inlet temp")
	checkTag(t, tags, "node.tag.sensor", "inlet")
	checkTag(t, tags, "node.tag.machine", "press-3")
	checkTag(t, tags, "node.tag.site", "plant-b")
	checkTag(t, tags, "node.tag.customer", "acme")
}

// TestTagCacheBoundaryStopsWalk verifies the boundary node's own tags are
// included and that ancestors above it are not consulted.
func TestTagCacheBoundaryStopsWalk(t *testing.T) {
	tree := map[string][]data.NodeEdge{
		"dev-root": {tagNode("dev-root", "device", "root",
			tagPt("customer", "acme"))},
		"press-3": {tagNode("press-3", "group", "dev-root",
			tagPt("machine", "press-3"))},
		"temp-1": {tagNode("temp-1", "modbusIo", "press-3")},
	}

	c := testTagCache([]string{data.PointTypeTag}, "press-3", tree)
	tags := resolve(t, c, "temp-1")

	checkTag(t, tags, "node.tag.machine", "press-3")
	if _, found := tags["node.tag.customer"]; found {
		t.Error("tag above the boundary node should not be inherited")
	}
}

// TestTagCacheTieBreak verifies that when two parents at the same depth
// define the same key, the node with the lowest ID wins deterministically.
func TestTagCacheTieBreak(t *testing.T) {
	tree := map[string][]data.NodeEdge{
		"a-press": {tagNode("a-press", "group", "root",
			tagPt("site", "plant-a"))},
		"b-press": {tagNode("b-press", "group", "root",
			tagPt("site", "plant-b"))},
		"temp-1": {
			tagNode("temp-1", "modbusIo", "b-press"),
			tagNode("temp-1", "modbusIo", "a-press"),
		},
	}

	c := testTagCache([]string{data.PointTypeTag}, "root", tree)
	tags := resolve(t, c, "temp-1")

	checkTag(t, tags, "node.tag.site", "plant-a")
}

// TestTagCacheAncestorTagEdit verifies the invalidation flow for a tag edit
// on an ancestor: the arriving point is recognized as a tag point, the cache
// is cleared, and the next resolution sees the new value.
func TestTagCacheAncestorTagEdit(t *testing.T) {
	tree := map[string][]data.NodeEdge{
		"press-3": {tagNode("press-3", "group", "root",
			tagPt("machine", "press-3"))},
		"temp-1": {tagNode("temp-1", "modbusIo", "press-3")},
	}

	c := testTagCache([]string{data.PointTypeTag}, "press-3", tree)
	tags := resolve(t, c, "temp-1")
	checkTag(t, tags, "node.tag.machine", "press-3")

	// the machine tag is edited on the ancestor
	edit := data.Points{tagPt("machine", "press-4")}
	if !c.hasTagPointType(edit) {
		t.Fatal("tag point edit not recognized as a tag point type")
	}
	tree["press-3"] = []data.NodeEdge{tagNode("press-3", "group", "root",
		tagPt("machine", "press-4"))}
	c.Clear()

	tags = resolve(t, c, "temp-1")
	checkTag(t, tags, "node.tag.machine", "press-4")
}

// TestTagCacheReparent verifies that after a node moves to a different
// parent and the cache is cleared (the edge-point handler's response), the
// node resolves the new parent's tags.
func TestTagCacheReparent(t *testing.T) {
	tree := map[string][]data.NodeEdge{
		"press-3": {tagNode("press-3", "group", "root",
			tagPt("machine", "press-3"))},
		"press-4": {tagNode("press-4", "group", "root",
			tagPt("machine", "press-4"))},
		"temp-1": {tagNode("temp-1", "modbusIo", "press-3")},
	}

	c := testTagCache([]string{data.PointTypeTag}, "root", tree)
	tags := resolve(t, c, "temp-1")
	checkTag(t, tags, "node.tag.machine", "press-3")

	tree["temp-1"] = []data.NodeEdge{tagNode("temp-1", "modbusIo", "press-4")}
	c.Clear()

	tags = resolve(t, c, "temp-1")
	checkTag(t, tags, "node.tag.machine", "press-4")
}

// TestTagCacheDescriptionRefresh verifies that points passed into Update
// refresh an existing cache entry (regression coverage for the stream path
// calling Update with an empty point set).
func TestTagCacheDescriptionRefresh(t *testing.T) {
	tree := map[string][]data.NodeEdge{
		"temp-1": {tagNode("temp-1", "modbusIo", "root",
			data.NewPointString(data.PointTypeDescription, "", "old name"))},
	}

	c := testTagCache([]string{data.PointTypeTag}, "root", tree)
	tags := resolve(t, c, "temp-1")
	checkTag(t, tags, "node.description", "old name")

	err := c.Update(nil, NewPoints{ID: "temp-1", Points: data.Points{
		data.NewPointString(data.PointTypeDescription, "", "new name"),
	}})
	if err != nil {
		t.Fatal("Error updating cache:", err)
	}

	tags = make(map[string]string)
	c.CopyTags("temp-1", tags)
	checkTag(t, tags, "node.description", "new name")
}

// TestTagCacheTombstone verifies a tombstoned local tag point removes the
// tag from an existing entry, and that after the cache clear it triggers,
// the inherited value from an ancestor shows through again.
func TestTagCacheTombstone(t *testing.T) {
	tree := map[string][]data.NodeEdge{
		"press-3": {tagNode("press-3", "group", "root",
			tagPt("site", "plant-a"))},
		"temp-1": {tagNode("temp-1", "modbusIo", "press-3",
			tagPt("site", "line-9"))},
	}

	c := testTagCache([]string{data.PointTypeTag}, "press-3", tree)
	tags := resolve(t, c, "temp-1")
	checkTag(t, tags, "node.tag.site", "line-9")

	// the local tag point is tombstoned
	tomb := tagPt("site", "line-9")
	tomb.Tombstone = 1

	// direct entry update removes the tag rather than leaving it stale
	err := c.Update(nil, NewPoints{ID: "temp-1", Points: data.Points{tomb}})
	if err != nil {
		t.Fatal("Error updating cache:", err)
	}
	tags = make(map[string]string)
	c.CopyTags("temp-1", tags)
	if _, found := tags["node.tag.site"]; found {
		t.Error("tombstoned tag should be removed from the entry")
	}

	// the tombstone is a tag point, so the db client clears the cache;
	// the next resolution inherits the ancestor's value
	if !c.hasTagPointType(data.Points{tomb}) {
		t.Fatal("tombstoned tag point not recognized as a tag point type")
	}
	tree["temp-1"] = []data.NodeEdge{tagNode("temp-1", "modbusIo", "press-3")}
	c.Clear()

	tags = resolve(t, c, "temp-1")
	checkTag(t, tags, "node.tag.site", "plant-a")
}

// TestTagCacheDiamond verifies an ancestor reachable by two paths is only
// applied once and resolution stays deterministic.
func TestTagCacheDiamond(t *testing.T) {
	tree := map[string][]data.NodeEdge{
		"dev-root": {tagNode("dev-root", "device", "root",
			tagPt("customer", "acme"))},
		"a-press": {tagNode("a-press", "group", "dev-root",
			tagPt("machine", "a-press"))},
		"b-press": {tagNode("b-press", "group", "dev-root")},
		"temp-1": {
			tagNode("temp-1", "modbusIo", "a-press"),
			tagNode("temp-1", "modbusIo", "b-press"),
		},
	}

	c := testTagCache([]string{data.PointTypeTag}, "dev-root", tree)
	tags := resolve(t, c, "temp-1")

	checkTag(t, tags, "node.tag.machine", "a-press")
	checkTag(t, tags, "node.tag.customer", "acme")
}
