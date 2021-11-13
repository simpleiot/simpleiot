package store

import (
	"crypto/md5"
	"encoding/binary"
	"sort"

	"github.com/simpleiot/simpleiot/data"
)

// updateHash updates the hash in all the upstream edges
// this is just a bad idea to be reaching back into data structures ...
func updateHash(node *data.Node, upEdges []*data.Edge, downEdges []*data.Edge) {
	// downstream edge hashes are used in the hash calculation, so sort them first
	sort.Sort(data.ByHash(downEdges))

	for _, up := range upEdges {
		h := md5.New()

		for _, p := range up.Points {
			d := make([]byte, 8)
			binary.LittleEndian.PutUint64(d, uint64(p.Time.UnixNano()))
			h.Write(d)
		}

		for _, p := range node.Points {
			d := make([]byte, 8)
			binary.LittleEndian.PutUint64(d, uint64(p.Time.UnixNano()))
			h.Write(d)
		}

		for _, downEdge := range downEdges {
			h.Write(downEdge.Hash)
		}

		up.Hash = h.Sum(nil)
	}
}

// calcHash calculates the hash for a node. downEdges should be sorted by
// hash before calling this function
func calcHash(node data.Node, upEdge data.Edge, downEdges []data.Edge) []byte {
	h := md5.New()

	// downstream edge hashes are used in the hash calculation, so sort them first

	for _, p := range upEdge.Points {
		d := make([]byte, 8)
		binary.LittleEndian.PutUint64(d, uint64(p.Time.UnixNano()))
		h.Write(d)
	}

	for _, p := range node.Points {
		d := make([]byte, 8)
		binary.LittleEndian.PutUint64(d, uint64(p.Time.UnixNano()))
		h.Write(d)
	}

	for _, downEdge := range downEdges {
		h.Write(downEdge.Hash)
	}

	return h.Sum(nil)
}
