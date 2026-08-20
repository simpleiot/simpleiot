package data

import (
	"fmt"
	"strings"
)

// The NATS server maps MQTT topics onto NATS subjects so that everything
// connected over NATS sees MQTT traffic. The rules below mirror the mapping
// the embedded server performs, so a client that wants to subscribe to an
// MQTT topic over NATS can work out the subject to use.
//
// See docs/user/mqtt.md for the user-facing description.

// MQTTTopicToSubject converts an MQTT topic name to the NATS subject the
// embedded broker publishes it on. Wildcards are not allowed in a topic name,
// so they are rejected here.
func MQTTTopicToSubject(topic string) (string, error) {
	return mqttToSubject(topic, false)
}

// MQTTFilterToSubject converts an MQTT topic filter to a NATS subject,
// mapping the MQTT wildcards + and # onto the NATS wildcards * and >.
func MQTTFilterToSubject(filter string) (string, error) {
	return mqttToSubject(filter, true)
}

// mqttToSubject converts an MQTT topic name or filter to a NATS subject:
//
//   - '/' becomes "/." when it is the first character, or follows a level that
//     already ended in '.'
//   - '/' becomes "./" when it is the last character or the next character is
//     also '/'
//   - '/' becomes '.' everywhere else
//   - '.' becomes "//"
//   - '+' becomes '*' and '#' becomes '>' in a filter
//
// Whitespace is not representable in a NATS subject, so a topic containing it
// is an error.
func mqttToSubject(topic string, wcOK bool) (string, error) {
	var b strings.Builder
	b.Grow(len(topic) + 8)

	endsWith := func(c byte) bool {
		s := b.String()
		return len(s) > 0 && s[len(s)-1] == c
	}

	last := len(topic) - 1

	for i := 0; i < len(topic); i++ {
		switch c := topic[i]; c {
		case '/':
			switch {
			case i == 0 || endsWith('.'):
				b.WriteString("/.")
			case i == last || topic[i+1] == '/':
				b.WriteString("./")
			default:
				b.WriteByte('.')
			}
		case '.':
			b.WriteString("//")
		case '+', '#':
			if !wcOK {
				return "", fmt.Errorf("wildcards are not allowed in the MQTT topic %q", topic)
			}
			if c == '+' {
				b.WriteByte('*')
			} else {
				b.WriteByte('>')
			}
		case ' ', '\t', '\n', '\r', '\f':
			return "", fmt.Errorf("MQTT topic %q contains whitespace, which a NATS subject cannot carry", topic)
		default:
			b.WriteByte(c)
		}
	}

	if endsWith('.') {
		b.WriteByte('/')
	}

	return b.String(), nil
}

// MQTTSubjectToTopic converts a NATS subject back to the MQTT topic it came
// from. It is the inverse of MQTTTopicToSubject and is how a client recovers
// the topic of a message delivered over NATS.
func MQTTSubjectToTopic(subject string) string {
	var b strings.Builder
	b.Grow(len(subject))

	last := len(subject) - 1

	for i := 0; i < len(subject); i++ {
		switch subject[i] {
		case '/':
			if i < last {
				switch subject[i+1] {
				case '.':
					b.WriteByte('/')
					i++
				case '/':
					b.WriteByte('.')
					i++
				}
			}
		case '.':
			b.WriteByte('/')
		default:
			b.WriteByte(subject[i])
		}
	}

	return b.String()
}
