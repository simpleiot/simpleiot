package data

import (
	"github.com/simpleiot/simpleiot/internal/pb"
	"google.golang.org/protobuf/proto"
)

// Message describes a notification that is sent to a particular user
type Message struct {
	ID             string
	UserID         string
	NotificationID string
	Email          string
	Phone          string
	Subject        string
	Message        string
}

// ToPb converts to protobuf data
func (m *Message) ToPb() ([]byte, error) {
	pbMsg := pb.Message{
		Id:             m.ID,
		UserId:         m.UserID,
		NotificationId: m.NotificationID,
		Email:          m.Email,
		Phone:          m.Phone,
		Subject:        m.Subject,
		Message:        m.Message,
	}

	return proto.Marshal(&pbMsg)
}

// PbDecodeMessage converts a protobuf to a message data structure
func PbDecodeMessage(data []byte) (Message, error) {
	pbMsg := &pb.Message{}

	err := proto.Unmarshal(data, pbMsg)
	if err != nil {
		return Message{}, err
	}

	return Message{
		ID:             pbMsg.Id,
		UserID:         pbMsg.UserId,
		NotificationID: pbMsg.NotificationId,
		Email:          pbMsg.Email,
		Phone:          pbMsg.Phone,
		Subject:        pbMsg.Subject,
		Message:        pbMsg.Message,
	}, nil
}
