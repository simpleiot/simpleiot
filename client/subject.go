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

// Destination indicates the destination for generated points, including the
// point type and key
type Destination struct {
	// NodeID indicating the destination for points; if not specified, the point
	// destination is determined by the Parent field
	NodeID string `point:"nodeID"`
	// Parent is set if points should be sent to the parent node; otherwise,
	// points are send to the origin node.
	Parent bool `point:"parent"`
	// HighRate indicates that points should be sent over the phrup NATS
	// subject. If set, points are never sent to the origin node; rather, it is
	// implied that points will be sent to the NodeID (if set) or the parent
	// node.
	HighRate bool `point:"highRate"`
	// PointType indicates the point type for generated points
	PointType string `point:"pointType"`
	// PointKey indicates the point key for generated points
	PointKey string `point:"pointKey"`
}

// Subject returns the NATS subject on which points for this Destination shall
// be published
func (sd Destination) Subject(originID string, parentID string) string {
	if sd.HighRate {
		// HighRate implies Parent
		destID := parentID
		if sd.NodeID != "" {
			destID = sd.NodeID
		}
		return fmt.Sprintf("phrup.%v.%v", destID, originID)
	}
	destID := originID
	if sd.NodeID != "" {
		destID = sd.NodeID
	} else if sd.Parent {
		destID = parentID
	}
	return SubjectNodePoints(destID)
}
