package client

// Device represents the instance SIOT is running on
type Device struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
}
