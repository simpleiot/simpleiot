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
	}

	pointAverager := NewTimeWindowAverager(3*time.Second, func(avg Point) {
		fmt.Println("Average (Value): ", avg.Value)
		fmt.Println()
		avgPoint = avg
	}, "hello")

	pointTicker := time.NewTicker(300 * time.Millisecond)
	startTime := time.Now()

	for time.Since(startTime) < time.Second*6 {
		<-pointTicker.C
		pointAverager.NewPoint(point)

		if avgPoint.Value != point.Value {
			t.Error("point avg is not correct")
		}
	}
}
