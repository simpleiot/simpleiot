package data

// TODO -- would like to move this to db/store package and make it internal

// Edge is used to describe the relationship
// between two nodes
type Edge struct {
	ID        string `json:"id"`
	Up        string `json:"up"`
	Down      string `json:"down"`
	Tombstone bool   `json:"tombstone"`
}
