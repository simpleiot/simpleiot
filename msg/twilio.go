package msg

import (
	"errors"

	"github.com/kevinburke/twilio-go"
)

// Twilio can be used to send messages through Twilio
type Twilio struct {
	twilioClient *twilio.Client
	smsFrom      string
}

// NewTwilio creates a new messanger object
func NewTwilio(twilioSid, twilioAuth, smsFrom string) *Twilio {
	return &Twilio{
		twilioClient: twilio.NewClient(twilioSid, twilioAuth, nil),
		smsFrom:      smsFrom,
	}
}

// SendSMS sends a sms message
func (m *Twilio) SendSMS(to, msg string) error {
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
