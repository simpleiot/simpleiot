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

func TestValidID(t *testing.T) {
	valid := []string{
		NewID(),
		"6f1c2b9e-4a3d-4e8f-9b21-7c5d0e3a8f42",
		"plant-07",
		"Line3",
		"root",
		strings.Repeat("a", 64),
	}
	for _, id := range valid {
		if err := ValidID(id); err != nil {
			t.Errorf("%q: %v", id, err)
		}
	}

	invalid := []string{
		"",
		strings.Repeat("a", 65),
		"-plant",
		"plant_07",
		"plant.07",
		"plant*",
		"plant>",
		"plant 07",
		"plänt",
	}
	for _, id := range invalid {
		if ValidID(id) == nil {
			t.Errorf("%q: accepted", id)
		}
	}
}
