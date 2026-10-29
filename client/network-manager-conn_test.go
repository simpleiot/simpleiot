//go:build linux

package client

import (
	"testing"

	"github.com/google/uuid"
)

func TestAssignConnUUIDs(t *testing.T) {
	const existing = "6f1c2b9e-4a3d-4e8f-9b21-7c5d0e3a8f42"
	conns := []NetworkManagerConn{
		{ID: "dwe2q7ja7n78z6s1fhegwemf88", UUID: existing},
		{ID: "a0b1c2d3e4f5g6h7j8k9m0n1p4"},
	}

	assigned := assignConnUUIDs(conns)

	if len(assigned) != 1 || assigned[0] != &conns[1] {
		t.Fatalf("assigned %v, want only the connection with no UUID", assigned)
	}
	if conns[0].UUID != existing {
		t.Errorf("existing UUID changed to %v", conns[0].UUID)
	}
	if _, err := uuid.Parse(conns[1].UUID); err != nil {
		t.Errorf("assigned UUID %q does not parse: %v", conns[1].UUID, err)
	}
	if len(assignConnUUIDs(conns)) != 0 {
		t.Error("second call assigned again")
	}
}
