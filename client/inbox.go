package client

// InboxPrefix is the reply inbox prefix for a connection belonging to id,
// which is a user node ID for a browser and a device credential's public
// key for a device or an enrolling instance.
//
// The default NATS inbox space, _INBOX.>, is shared by every client on a
// server, so a connection allowed to subscribe to it receives every other
// client's replies. A connection instead gets a prefix of its own, granted
// to it and to nothing else, and sets nats.CustomInboxPrefix to match.
func InboxPrefix(id string) string {
	return "_INBOX_" + id
}
