package msg

import (
	"errors"

	"github.com/kevinburke/twilio-go"
)

// Messenger can be used to send various types of messages
type Messenger struct {
	twilioClient *twilio.Client
	smsFrom      string
}

// NewMessenger creates a new messanger object
func NewMessenger(twilioSid, twilioAuth, smsFrom string) *Messenger {
	return &Messenger{
		twilioClient: twilio.NewClient(twilioSid, twilioAuth, nil),
		smsFrom:      smsFrom,
	}
}

// SendSMS sends a sms message
func (m *Messenger) SendSMS(to, msg string) error {
	if m.twilioClient == nil {
		return errors.New("Twilio not set up")
	}

	ret, err := m.twilioClient.Messages.SendMessage(m.smsFrom, to, msg, nil)
	if err != nil {
		return err
	}

	if ret.ErrorCode != 0 {
		return errors.New(ret.ErrorMessage)
	}

	return nil
}
