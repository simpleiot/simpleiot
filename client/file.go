package client

// File represents a file that a user uploads or is present in some location
type File struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	Name        string `point:"name"`
	Data        string `point:"data"`
	Binary      bool   `point:"binary"`
}
