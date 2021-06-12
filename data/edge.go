package data

// TODO -- would like to move this to db/store package and make it internal

// Edge is used to describe the relationship
// between two nodes
type Edge struct {
	ID     string `json:"id"`
	Up     string `json:"up"`
	Down   string `json:"down"`
	Points Points `json:"points"`
}

// IsTombstone returns true of edge points to a deleted node
func (e *Edge) IsTombstone() bool {
	tombstone, _ := e.Points.ValueBool("", PointTypeTombstone, 0)
	return tombstone
}
