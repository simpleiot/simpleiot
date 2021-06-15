package data

import "strings"

// TODO -- would like to move this to db/store package and make it internal

// Edge is used to describe the relationship
// between two nodes
type Edge struct {
	ID     string `json:"id"`
	Up     string `json:"up"`
	Down   string `json:"down"`
	Points Points `json:"points"`
	Hash   []byte `json:"hash"`
}

// IsTombstone returns true of edge points to a deleted node
func (e *Edge) IsTombstone() bool {
	tombstone, _ := e.Points.ValueBool("", PointTypeTombstone, 0)
	return tombstone
}

// ByEdgeID implements sort interface for NodeEdge by ID
type ByEdgeID []Edge

func (a ByEdgeID) Len() int           { return len(a) }
func (a ByEdgeID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByEdgeID) Less(i, j int) bool { return strings.Compare(a[i].ID, a[j].ID) < 0 }
