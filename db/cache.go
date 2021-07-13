package db

import (
	"fmt"

	"github.com/genjidb/genji"
	"github.com/simpleiot/simpleiot/data"
)

// The following contains node with all its edges
type nodeAndEdges struct {
	node *data.Node
	up   []*data.Edge
	down []*data.Edge
}

type nodeEdgeCache struct {
	nodes map[string]*nodeAndEdges
	edges map[string]*data.Edge
	tx    *genji.Tx
}

func newNodeEdgeCache(tx *genji.Tx) *nodeEdgeCache {
	return &nodeEdgeCache{
		nodes: make(map[string]*nodeAndEdges),
		edges: make(map[string]*data.Edge),
		tx:    tx,
	}
}

// this function builds a cache of edges and replaces
// the edge in the array with the one in the cache if present
// this ensures the edges in the cache are the same as the ones
// in the array. The edges parameter may be modified.
func (nec *nodeEdgeCache) cacheEdges(edges []*data.Edge) {
	for i, e := range edges {
		eCache, ok := nec.edges[e.ID]
		if !ok {
			nec.edges[e.ID] = e
		} else {
			edges[i] = eCache
		}
	}
}

// this function gets a node, all its edges, and caches it
func (nec *nodeEdgeCache) getNodeAndEdges(id string) (*nodeAndEdges, error) {
	ret, ok := nec.nodes[id]
	if ok {
		return ret, nil
	}

	ret = &nodeAndEdges{}

	node, err := txNode(nec.tx, id)
	if err != nil {
		return ret, err
	}

	downEdges, err := txEdgeDown(nec.tx, id)
	if err != nil {
		return ret, err
	}

	nec.cacheEdges(downEdges)

	upEdges, err := txEdgeUp(nec.tx, id, true)
	if err != nil {
		return ret, err
	}

	nec.cacheEdges(upEdges)

	ret.node = node
	ret.up = upEdges
	ret.down = downEdges

	nec.nodes[id] = ret

	return ret, nil
}

// populate cache and update hashes for node and edges all the way up to root, and one level down from current node
func (nec *nodeEdgeCache) processNode(ne *nodeAndEdges, newEdge bool) error {
	updateHash(ne.node, ne.up, ne.down)
	for _, upEdge := range ne.up {
		if upEdge.Up == "" || upEdge.Up == "none" {
			continue
		}

		neUp, err := nec.getNodeAndEdges(upEdge.Up)

		if err != nil {
			return fmt.Errorf("Error getting neUp: %w", err)
		}

		if newEdge {
			neUp.down = append(neUp.down, upEdge)
		}

		err = nec.processNode(neUp, false)

		if err != nil {
			return fmt.Errorf("Error processing node to update hash: %w", err)
		}
	}

	return nil
}

func (nec *nodeEdgeCache) writeEdges() error {
	//fmt.Println("CLIFF: edges: ", nec.edges)
	for _, e := range nec.edges {
		err := nec.tx.Exec(`insert into edges values ? on conflict do replace`, e)

		if err != nil {
			return fmt.Errorf("Error updating hash in edge %v: %v", e.ID, err)
		}
	}

	return nil
}
