package client

import "testing"

func TestShellyAddr(t *testing.T) {
	for _, test := range []struct {
		ip   string
		want string
		ok   bool
	}{
		{"192.168.1.42", "192.168.1.42", true},
		{"192.168.1.42:8080", "192.168.1.42:8080", true},
		{"127.0.0.1:41234", "127.0.0.1:41234", true},
		{"fe80::1", "[fe80::1]", true},
		{"[fe80::1]:80", "[fe80::1]:80", true},
		{"", "", false},
		{"shelly.local", "", false},
		{"192.168.1.42/rpc", "", false},
		{"192.168.1.42:notaport", "", false},
		{"192.168.1.42:99999", "", false},
		{"evil.example.com@192.168.1.42", "", false},
	} {
		got, err := shellyAddr(test.ip)
		if test.ok && (err != nil || got != test.want) {
			t.Errorf("%q: got %q, %v; want %q", test.ip, got, err, test.want)
		}
		if !test.ok && err == nil {
			t.Errorf("%q: expected error, got %q", test.ip, got)
		}
	}
}

func TestShellyIdentityMatches(t *testing.T) {
	di := shellyDeviceInfo{ID: "shellyplusplugus-b0b21c12ad58", MAC: "B0B21C12AD58"}

	if !shellyIdentityMatches(di, "B0B21C12AD58") {
		t.Error("same MAC did not match")
	}
	if !shellyIdentityMatches(di, "b0b21c12ad58") {
		t.Error("MAC comparison is case sensitive")
	}
	if shellyIdentityMatches(di, "C049EF8889A0") {
		t.Error("different MAC matched")
	}
	if shellyIdentityMatches(shellyDeviceInfo{}, "") {
		t.Error("device with no MAC matched an empty id")
	}
}
