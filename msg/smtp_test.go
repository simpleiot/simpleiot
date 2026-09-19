package msg

import "testing"

func TestHeaderValue(t *testing.T) {
	for _, test := range []struct {
		in, want string
	}{
		{"plain subject", "plain subject"},
		{"alarm\r\nBcc: attacker@example.com", "alarm  Bcc: attacker@example.com"},
		{"alarm\nX-Injected: 1", "alarm X-Injected: 1"},
		{"joe@example.com\r", "joe@example.com "},
		{"", ""},
	} {
		if got := headerValue(test.in); got != test.want {
			t.Errorf("%q: got %q, want %q", test.in, got, test.want)
		}
	}
}
