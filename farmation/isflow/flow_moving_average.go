package isflow

import (
	"fmt"
	"math"

	movingaverage "github.com/RobinUS2/golang-moving-average"
)

// FlowMovAvg stores fields used to apply moving averages
// to smooth incoming flow data
type FlowMovAvg struct {
	// Moving average windows
	winLong  int
	winShort int
	// Moving averagers
	movAvgLong  *movingaverage.MovingAverage
	movAvgShort *movingaverage.MovingAverage
	// Allowable percent difference between flow rates
	// from long and short moving averages
	percentDiff int
}

// Define values to reference different values by
const (
	WindowLong int = iota
	WindowShort
	PercentDiff
)

// NewFlowMovAvg intitializes a new pointer to a FlowMovAvg type
func NewFlowMovAvg(winLong, winShort, percentDiff int) FlowMovAvg {
	return FlowMovAvg{
		winLong:     winLong,
		winShort:    winShort,
		movAvgLong:  movingaverage.New(winLong),
		movAvgShort: movingaverage.New(winShort),
		percentDiff: percentDiff,
	}
}

// AddDataPoint adds a new flow rate data point to the
// moving averages and returns the new flow rate, min, and max
func (f FlowMovAvg) AddDataPoint(data float64) (avg, min, max, avgShort float64) {
	f.movAvgLong.Add(data)
	f.movAvgShort.Add(data)

	avg = f.movAvgLong.Avg()
	min, _ = f.movAvgLong.Min()
	max, _ = f.movAvgLong.Max()
	avgShort = f.movAvgShort.Avg()

	// FIXME
	fmt.Println("Long:", f.movAvgLong.Count(), f.winLong)
	fmt.Println("Short:", f.movAvgShort.Count(), f.winShort)
	fmt.Println("PercentDiff:", f.percentDiff)

	// We are using the short-window moving average
	// to track instantaneous change. If the instantaneous
	// change is big enough, we reset the long-window
	// moving avg to the short avg in order provide a
	// shorter response time to the user.
	acceptedVal := (avg + avgShort) / 2
	calculatedPercentDiff := int(math.Abs(avgShort-avg) / acceptedVal * 100)
	if calculatedPercentDiff >= f.percentDiff {
		// Return the value from the short-window
		// averager
		avg = avgShort
		min, _ = f.movAvgShort.Min()
		max, _ = f.movAvgShort.Max()
		// Reset the long-window averager
		f.movAvgLong = movingaverage.New(f.winLong)
		// Add the current data point back to the long
		// average to start it off.
		f.movAvgLong.Add(data)
	}

	return avg, min, max, avgShort
}

// UpdateReset updates the specified field to the given value
// If the field is an averager window, UpdateReset resets the
// averager
func (f FlowMovAvg) UpdateReset(whichVal, val int) {
	// FIXME
	fmt.Println("Update/Reset")
	switch whichVal {
	case WindowLong:
		fmt.Println("hello")
		f.movAvgLong = movingaverage.New(val)
		f.winLong = val
	case WindowShort:
		f.movAvgShort = movingaverage.New(val)
		f.winShort = val
	case PercentDiff:
		f.percentDiff = val
	}
}
