package data

import "testing"

func TestMQTTTopicToSubject(t *testing.T) {
	tests := []struct {
		topic   string
		subject string
	}{
		{"a/b/c", "a.b.c"},
		{"foo", "foo"},
		{"a.b", "a//b"},
		{"spBv1.0/plant/DDATA/line3/tank", "spBv1//0.plant.DDATA.line3.tank"},
		{"/a", "/.a"},
		{"a/", "a./"},
		{"foo//bar", "foo./.bar"},
	}

	for _, test := range tests {
		s, err := MQTTTopicToSubject(test.topic)
		if err != nil {
			t.Errorf("%v: unexpected error: %v", test.topic, err)
			continue
		}

		if s != test.subject {
			t.Errorf("%v: got subject %v, expected %v", test.topic, s, test.subject)
		}

		topic := MQTTSubjectToTopic(test.subject)
		if topic != test.topic {
			t.Errorf("%v: converting back gave %v", test.subject, topic)
		}
	}
}

func TestMQTTTopicToSubjectErrors(t *testing.T) {
	for _, topic := range []string{"a/+/c", "a/#", "a b"} {
		if _, err := MQTTTopicToSubject(topic); err == nil {
			t.Errorf("%v: expected an error", topic)
		}
	}
}

func TestMQTTFilterToSubject(t *testing.T) {
	tests := []struct {
		filter  string
		subject string
	}{
		{"a/b/c", "a.b.c"},
		{"a/+/c", "a.*.c"},
		{"a/#", "a.>"},
		{"#", ">"},
		{"+", "*"},
		{"spBv1.0/#", "spBv1//0.>"},
	}

	for _, test := range tests {
		s, err := MQTTFilterToSubject(test.filter)
		if err != nil {
			t.Errorf("%v: unexpected error: %v", test.filter, err)
			continue
		}

		if s != test.subject {
			t.Errorf("%v: got subject %v, expected %v", test.filter, s, test.subject)
		}
	}
}
