package nats

// create subject strings for various types of messages

// SubjectNodePoints constructs a NATS subject for node points
func SubjectNodePoints(nodeID string) string {
	return "node." + nodeID + ".points"
}

// SubjectNodeAllPoints provides subject for all points for any node
func SubjectNodeAllPoints() string {
	return "node.*.points"
}
