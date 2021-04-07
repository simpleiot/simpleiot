package data

import (
	"github.com/simpleiot/simpleiot/internal/pb"
	"google.golang.org/protobuf/proto"
)

// Notification represents a message sent by a node
type Notification struct {
	ID         string
	SourceNode string
	Subject    string
	Message    string
}

// ToPb converts to protobuf data
func (n *Notification) ToPb() ([]byte, error) {
	pbNot := pb.Notification{
		Id:         n.ID,
		SourceNode: n.SourceNode,
		Subject:    n.Subject,
		Msg:        n.Message,
	}

	return proto.Marshal(&pbNot)
}

// PbDecodeNotification converts a protobuf to notification data structure
func PbDecodeNotification(data []byte) (Notification, error) {
	pbNot := &pb.Notification{}

	err := proto.Unmarshal(data, pbNot)
	if err != nil {
		return Notification{}, err
	}

	return Notification{
		ID:         pbNot.Id,
		SourceNode: pbNot.SourceNode,
		Subject:    pbNot.Subject,
		Message:    pbNot.Msg,
	}, nil
}
