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
	if avgPoint.Val() != 0 {
		t.Error("point avg with 0 points is not correct: ", avgPoint.Val())
	}

	// Round 1.5
	feedPoints(pointAverager, avg)

	avgPoint = pointAverager.GetAverage()
	if avgPoint.Val() != avg {
		t.Error("point avg is not correct")
	}

	pointAverager.ResetAverage()

	// Round 2
	avg = 2

	pointAverager = NewPointAverager("testPoint")
	feedPoints(pointAverager, avg)

	avgPoint = pointAverager.GetAverage()
	if avgPoint.Val() != avg {
		t.Error("point avg is not correct")
	}

	pointAverager.ResetAverage()

	// Round 3
	avg = 200.00

	pointAverager = NewPointAverager("testPoint")
	feedPoints(pointAverager, avg)

	avgPoint = pointAverager.GetAverage()
	if avgPoint.Val() != avg {
		t.Error("point avg is not correct")
	}
}

func feedPoints(pointAverager *PointAverager, avg float64) {
	point := NewPointFloat("", "", avg-100)
	point.Time = time.Now()
	pointAverager.AddPoint(point)
	pointAverager.AddPoint(point)

	point = NewPointFloat("", "", avg)
	point.Time = time.Now()
	pointAverager.AddPoint(point)
	pointAverager.AddPoint(point)

	point = NewPointFloat("", "", avg+100)
	point.Time = time.Now()
	pointAverager.AddPoint(point)
	pointAverager.AddPoint(point)
}
