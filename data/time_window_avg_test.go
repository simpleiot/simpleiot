package data

import (
	"fmt"
	"testing"
	"time"
)

func TestTimeWindowAverager(t *testing.T) {
	sample := Sample{
		Time:  time.Now(),
		Value: 200,
		Min:   100,
		Max:   300,
	}

	sampleAverager := NewTimeWindowAverager(600*time.Millisecond, func(avg Sample) {
		fmt.Println("running callback")
		if avg.Value != sample.Value {
			t.Error("sample avg is not correct")
		}
		if avg.Min != sample.Min {
			t.Error("sample min is not correct")
		}
		if avg.Max != sample.Max {
			t.Error("sample max is not correct")
		}
	}, "hello")

	sampleTicker := time.NewTicker(300 * time.Millisecond)
	startTime := time.Now()

	for time.Since(startTime) < 1220*time.Millisecond {
		select {
		case <-sampleTicker.C:
			fmt.Println("new sample")
			sampleAverager.NewSample(sample)
		}
	}
}
