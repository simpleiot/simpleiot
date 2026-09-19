package client

import "github.com/simpleiot/simpleiot/data"

// ScanNodes exposes scanHelper to the external tests.
func (m *Manager[T]) ScanNodes(id string) ([]data.NodeEdge, error) {
	return m.scanHelper(id, nil)
}
