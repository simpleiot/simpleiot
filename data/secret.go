package data

// secretPointTypes lists the point types that hold a credential: a user's
// password hash, the tokens a sync, message service, database, or Particle
// client presents to another service, a Wi-Fi pre-shared key, and an
// enrollment token. They are stored like any other point, so the places
// that hand nodes to a client (node replies, exports, the UI) leave their
// values out unless secrets are asked for explicitly.
var secretPointTypes = map[string]bool{
	PointTypePass:        true,
	PointTypeAuthToken:   true,
	PointTypeEnrollToken: true,
	PointTypeSID:         true,
	PointTypePSK:         true,
}

// IsSecretPointType reports whether points of this type hold a credential
// that should not be handed out with the rest of a node.
func IsSecretPointType(typ string) bool {
	return secretPointTypes[typ]
}

// SecretPointTypes returns the point types IsSecretPointType reports, for
// documentation and messages.
func SecretPointTypes() []string {
	out := make([]string, 0, len(secretPointTypes))
	for t := range secretPointTypes {
		out = append(out, t)
	}
	return out
}

// RedactNodes strips the value of every secret point from a set of nodes
// before they are handed to a browser or an API client. The point stays,
// with an empty value and its timestamp, so a UI can show that a
// credential is set without receiving it; a write of the same point
// replaces the stored value as usual. The server's own clients read nodes
// on the plain NATS subject and are not affected.
func RedactNodes(nodes []NodeEdge) []NodeEdge {
	for i := range nodes {
		nodes[i].Points = RedactPoints(nodes[i].Points)
	}
	return nodes
}

// RedactPoints is RedactNodes for one set of points.
func RedactPoints(pts Points) Points {
	out := make(Points, 0, len(pts))
	for _, p := range pts {
		if IsSecretPointType(p.Type) && p.Txt() != "" {
			p.PutString("")
		}
		out = append(out, p)
	}
	return out
}
