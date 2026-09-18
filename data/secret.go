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
