package data

import (
	"fmt"
	"testing"
	"time"
)

func TestNewPoint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	var avgPoint Point

	point := NewPointFloat("", "", 200)
	point.Time = time.Now()

	pointAverager := NewTimeWindowAverager(3*time.Second, func(avg Point) {
		fmt.Println("Average (Value): ", avg.Val())
		fmt.Println()
		avgPoint = avg
	}, "hello")

	pointTicker := time.NewTicker(300 * time.Millisecond)
	startTime := time.Now()

	for time.Since(startTime) < time.Second*6 {
		<-pointTicker.C
		pointAverager.NewPoint(point)

		if avgPoint.Val() != point.Val() {
			t.Error("point avg is not correct")
		}
	}
}
