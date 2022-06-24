package client

// Manager manages a node type, watches for changes, adds/removes instances that get
// added/deleted
type Manager[T any] struct {
}

// NewManager ...
func NewManager[T any](parent string, construct func(config T) Client) *Manager[T] {
	return &Manager[T]{}
}
