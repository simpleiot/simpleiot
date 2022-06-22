package node

// ClientManager manages a node type, watches for changes, adds/removes instances that get
// added/deleted
type ClientManager[T any] struct {
}

// NewClientManager ...
func NewClientManager[T any](parent string, construct func(config T) Client) *ClientManager[T] {
	return &ClientManager[T]{}
}
