package client

import (
	"math"
	"testing"
	"time"
)

func TestPointDuration(t *testing.T) {
	def := 3 * time.Second
	floor := 10 * time.Millisecond

	tests := []struct {
		name string
		v    float64
		want time.Duration
	}{
		{"unset", 0, def},
		{"negative", -5, def},
		{"nan", math.NaN(), def},
		{"normal", 250, 250 * time.Millisecond},
		{"below floor", 1, floor},
		{"overflow", 1e30, maxPointDuration},
		{"infinite", math.Inf(1), maxPointDuration},
	}

	for _, tt := range tests {
		got := pointDuration(tt.v, time.Millisecond, def, floor)
		if got != tt.want {
			t.Errorf("%v: pointDuration(%v) = %v, want %v", tt.name, tt.v, got, tt.want)
		}
	}

	// a zero default is what a client that only polls when asked uses
	if got := pointDuration(0, time.Millisecond, 0, floor); got != 0 {
		t.Errorf("zero default: got %v, want 0", got)
	}
}
