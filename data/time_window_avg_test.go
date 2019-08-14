package data

import (
	"fmt"
	"testing"
	"time"
)

func TestNewSample(t *testing.T) {
	var avgSample Sample

	sample := Sample{
		Time:  time.Now(),
		Value: 200,
		Min:   100,
		Max:   300,
	}

	sampleAverager := NewTimeWindowAverager(3*time.Second, func(avg Sample) {
		fmt.Println("Average (Value): ", avg.Value)
		fmt.Println("Min:             ", avg.Min)
		fmt.Println("Max:             ", avg.Max)
		fmt.Println()
		avgSample = avg
	}, "hello")

	sampleTicker := time.NewTicker(300 * time.Millisecond)
	startTime := time.Now()

	for time.Since(startTime) < time.Second*6 {
		select {
		case <-sampleTicker.C:
			sampleAverager.NewSample(sample)

			if avgSample.Value != sample.Value {
				t.Error("sample avg is not correct")
			}
			if avgSample.Min != sample.Min {
				t.Error("sample min is not correct")
			}
			if avgSample.Max != sample.Max {
				t.Error("sample max is not correct")
			}
		}
	}
}
