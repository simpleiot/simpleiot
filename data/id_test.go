package data

import (
	"strings"
	"testing"
)

func TestNewID(t *testing.T) {
	const alphabet = "0123456789abcdefghjkmnpqrstvwxyz"
	seen := make(map[string]bool)
	for range 5000 {
		id := NewID()
		if len(id) != 26 {
			t.Fatalf("%q: length %v, want 26", id, len(id))
		}
		for _, c := range id {
			if !strings.ContainsRune(alphabet, c) {
				t.Fatalf("%q: %q is not in the alphabet", id, c)
			}
		}
		if !strings.ContainsRune("048cgmrw", rune(id[25])) {
			t.Fatalf("%q: last character carries the leftover bits", id)
		}
		if seen[id] {
			t.Fatalf("%q generated twice", id)
		}
		seen[id] = true
	}
}
