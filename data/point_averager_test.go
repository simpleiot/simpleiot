package data

import (
	"testing"
	"time"
)

func TestPointAverager(t *testing.T) {
	// Round 1
	avg := 2000.00

	pointAverager := NewPointAverager("testPoint")
	avgPoint := pointAverager.GetAverage()
	if avgPoint.Value != 0 {
		t.Error("point avg with 0 points is not correct: ", avgPoint.Value)
	}

	// Round 1.5
	feedPoints(pointAverager, avg)

	avgPoint = pointAverager.GetAverage()
	if avgPoint.Value != avg {
		t.Error("point avg is not correct")
	}

	pointAverager.ResetAverage()

	// Round 2
	avg = 2

	pointAverager = NewPointAverager("testPoint")
	feedPoints(pointAverager, avg)

	avgPoint = pointAverager.GetAverage()
	if avgPoint.Value != avg {
		t.Error("point avg is not correct")
	}

	pointAverager.ResetAverage()

	// Round 3
	avg = 200.00

	pointAverager = NewPointAverager("testPoint")
	feedPoints(pointAverager, avg)

	avgPoint = pointAverager.GetAverage()
	if avgPoint.Value != avg {
		t.Error("point avg is not correct")
	}
}

func feedPoints(pointAverager *PointAverager, avg float64) {
	point := Point{
		Time:  time.Now(),
		Value: avg - 100,
	}
	pointAverager.AddPoint(point)
	pointAverager.AddPoint(point)

	point = Point{
		Time:  time.Now(),
		Value: avg,
	}
	pointAverager.AddPoint(point)
	pointAverager.AddPoint(point)

	point = Point{
		Time:  time.Now(),
		Value: avg + 100,
	}
	pointAverager.AddPoint(point)
	pointAverager.AddPoint(point)
}
