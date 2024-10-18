package client

// Group represents a group node
type Group struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
}
