package store

/* old implementation

// updateHash updates the hash in all the upstream edges
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

*/
