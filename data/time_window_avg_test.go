package data

import (
	"fmt"
	"testing"
	"time"
)

func TestNewPoint(t *testing.T) {
	var avgPoint Point

	point := Point{
		Time:  time.Now(),
		Value: 200,
		Min:   100,
		Max:   300,
	}

	pointAverager := NewTimeWindowAverager(3*time.Second, func(avg Point) {
		fmt.Println("Average (Value): ", avg.Value)
		fmt.Println("Min:             ", avg.Min)
		fmt.Println("Max:             ", avg.Max)
		fmt.Println()
		avgPoint = avg
	}, "hello")

	pointTicker := time.NewTicker(300 * time.Millisecond)
	startTime := time.Now()

	for time.Since(startTime) < time.Second*6 {
		select {
		case <-pointTicker.C:
			pointAverager.NewPoint(point)

			if avgPoint.Value != point.Value {
				t.Error("point avg is not correct")
			}
			if avgPoint.Min != point.Min {
				t.Error("point min is not correct")
			}
			if avgPoint.Max != point.Max {
				t.Error("point max is not correct")
			}
		}
	}
}
