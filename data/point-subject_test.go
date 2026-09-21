package data

import (
	"fmt"
	"strings"
	"testing"
)

func TestPointCheckSubjectTokens(t *testing.T) {
	tests := []struct {
		desc string
		p    Point
		ok   bool
	}{
		{"plain point", Point{Type: "temp", Key: "cpu"}, true},
		{"no key", Point{Type: "temp"}, true},
		{"key with underscores and dashes", Point{Type: "temp", Key: "pcie_bridge-1"}, true},
		// the case that sent points to the wrong client handler: the
		// subject gains a token and everything after it shifts
		{"period in key", Point{Type: "temp", Key: "PCIe_bridge_0.95V"}, false},
		{"period in type", Point{Type: "temp.board", Key: "cpu"}, false},
		{"space in key", Point{Type: "temp", Key: "board area"}, false},
		{"wildcard in key", Point{Type: "temp", Key: "board*"}, false},
		{"wildcard in type", Point{Type: ">", Key: "cpu"}, false},
		// a corrupted serial read stored subjects like this one, and NATS
		// then refused the stream index on every restart
		{"NUL in type", Point{Type: "cha\x00\x00rgerState", Key: "0"}, false},
		{"NUL in key", Point{Type: "temp", Key: "cpu\x00"}, false},
		{"invalid UTF-8 in key", Point{Type: "temp", Key: "cpu\xff"}, false},
		{"replacement character in type", Point{Type: "temp\uFFFD", Key: "cpu"}, false},
		{"non-ASCII UTF-8 in key", Point{Type: "temp", Key: "zone-ä"}, true},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := test.p.CheckSubjectTokens()

			if test.ok && err != nil {
				t.Fatalf("Expected %v to be accepted, got: %v", test.p, err)
			}

			if !test.ok {
				if err == nil {
					t.Fatalf("Expected %v to be rejected", test.p)
				}

				// the sender has to be able to find the point from the
				// message alone, so it names the offending value
				offending := test.p.Key
				if _, bad := invalidTokenChar(test.p.Type); bad {
					offending = test.p.Type
				}

				if !strings.Contains(err.Error(), fmt.Sprintf("%q", offending)) {
					t.Errorf("Expected error to name %q, got: %v", offending, err)
				}
			}
		})
	}
}

func TestSubjectSafeToken(t *testing.T) {
	tests := []struct {
		in  string
		exp string
	}{
		{"cpu0", "cpu0"},
		{"devfreq-17000000.gpu", "devfreq-17000000_gpu"},
		{"eth0.100", "eth0_100"},
		{"/boot/efi", "/boot/efi"},
		{"board area", "board_area"},
		{"board\x00area", "board_area"},
		{"board\xffarea", "board_area"},
		{"", ""},
	}

	for _, test := range tests {
		got := SubjectSafeToken(test.in)

		if got != test.exp {
			t.Errorf("SubjectSafeToken(%q) = %q, expected %q", test.in, got, test.exp)
		}

		if err := (Point{Type: "test", Key: got}).CheckSubjectTokens(); err != nil {
			t.Errorf("SubjectSafeToken(%q) produced a key that is still rejected: %v",
				test.in, err)
		}
	}
}
