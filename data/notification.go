package data

// Notification represents a message sent by a node
type Notification struct {
	ID         string
	SourceNode string
	Subject    string
	Message    string
}
