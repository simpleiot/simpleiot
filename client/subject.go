package client

import "fmt"

// create subject strings for various types of messages

// SubjectNodePoints constructs a NATS subject for node points
func SubjectNodePoints(nodeID string) string {
	return fmt.Sprintf("p.%v", nodeID)
}

// SubjectEdgePoints constructs a NATS subject for edge points
func SubjectEdgePoints(nodeID, parentID string) string {
	return fmt.Sprintf("p.%v.%v", nodeID, parentID)
}

// SubjectNodeAllPoints provides subject for all points for any node
func SubjectNodeAllPoints() string {
	return "p.*"
}

// SubjectEdgeAllPoints provides subject for all edge points for any node
func SubjectEdgeAllPoints() string {
	return "p.*.*"
}

// SubjectNodeHRPoints constructs a NATS subject for high rate node points
func SubjectNodeHRPoints(nodeID string) string {
	return fmt.Sprintf("phr.%v", nodeID)
}
