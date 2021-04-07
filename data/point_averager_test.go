package data

import (
	"testing"
	"time"
)

func TestPointAverager(t *testing.T) {
	// Round 1
	min := 50.01
	max := 5000.90
	avg := 2000.00

	pointAverager := NewPointAverager("testPoint")
	avgPoint := pointAverager.GetAverage()
	if avgPoint.Value != 0 {
		t.Error("point avg with 0 points is not correct: ", avgPoint.Value)
	}

	// Round 1.5
	feedPoints(pointAverager, avg, min, max)

	avgPoint = pointAverager.GetAverage()
	if avgPoint.Value != avg {
		t.Error("point avg is not correct")
	}
	if avgPoint.Min != min {
		t.Error("point min is not correct")
	}
	if avgPoint.Max != max {
		t.Error("point max is not correct")
	}

	pointAverager.ResetAverage()

	// Round 2
	min = 1
	max = 3
	avg = 2

	pointAverager = NewPointAverager("testPoint")
	feedPoints(pointAverager, avg, min, max)

	avgPoint = pointAverager.GetAverage()
	if avgPoint.Value != avg {
		t.Error("point avg is not correct")
	}
	if avgPoint.Min != min {
		t.Error("point min is not correct")
	}
	if avgPoint.Max != max {
		t.Error("point max is not correct")
	}

	pointAverager.ResetAverage()

	// Round 3
	min = 5.01
	max = 500.90
	avg = 200.00

	pointAverager = NewPointAverager("testPoint")
	feedPoints(pointAverager, avg, min, max)

	avgPoint = pointAverager.GetAverage()
	if avgPoint.Value != avg {
		t.Error("point avg is not correct")
	}
	if avgPoint.Min != min {
		t.Error("point min is not correct")
	}
	if avgPoint.Max != max {
		t.Error("point max is not correct")
	}
}

func feedPoints(pointAverager *PointAverager, avg, min, max float64) {
	point := Point{
		Time:  time.Now(),
		Value: avg - 100,
		Min:   min + 100,
		Max:   max - 100,
	}
	pointAverager.AddPoint(point)
	pointAverager.AddPoint(point)

	point = Point{
		Time:  time.Now(),
		Value: avg,
		Min:   min,
		Max:   max,
	}
	pointAverager.AddPoint(point)
	pointAverager.AddPoint(point)

	point = Point{
		Time:  time.Now(),
		Value: avg + 100,
		Min:   min + 100,
		Max:   max - 100,
	}
	pointAverager.AddPoint(point)
	pointAverager.AddPoint(point)
}
