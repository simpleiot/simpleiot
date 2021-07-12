package nats

import "fmt"

// create subject strings for various types of messages

// SubjectNodePoints constructs a NATS subject for node points
func SubjectNodePoints(nodeID string) string {
	return fmt.Sprintf("node.%v.points", nodeID)
}

// SubjectEdgePoints constructs a NATS subject for edge points
func SubjectEdgePoints(nodeID, parentID string) string {
	return fmt.Sprintf("node.%v.%v.points", nodeID, parentID)
}

// SubjectNodeAllPoints provides subject for all points for any node
func SubjectNodeAllPoints() string {
	return "node.*.points"
}
