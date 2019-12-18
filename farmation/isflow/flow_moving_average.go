package isflow

import (
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
	// User-Settable sample duration
	sampleDuration int
}

// Define values to reference different values by
const (
	WindowLong int = iota
	WindowShort
	PercentDiff
	SampleDuration
)

// NewFlowMovAvg intitializes a new pointer to a FlowMovAvg type
func NewFlowMovAvg(winLong, winShort, percentDiff, sampleDuration int) *FlowMovAvg {

	winLong = forceMultiple(winLong, sampleDuration)
	winShort = forceMultiple(winShort, sampleDuration)

	return &FlowMovAvg{
		winLong:        winLong,
		winShort:       winShort,
		movAvgLong:     movingaverage.New(winLong / sampleDuration),
		movAvgShort:    movingaverage.New(winShort / sampleDuration),
		percentDiff:    percentDiff,
		sampleDuration: sampleDuration,
	}
}

// AddDataPoint adds a new flow rate data point to the
// moving averages and returns the new flow rate, min, and max
func (f *FlowMovAvg) AddDataPoint(data float64) (avg, min, max, avgShort float64) {
	f.movAvgLong.Add(data)
	f.movAvgShort.Add(data)

	avg = f.movAvgLong.Avg()
	min, _ = f.movAvgLong.Min()
	max, _ = f.movAvgLong.Max()
	avgShort = f.movAvgShort.Avg()

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
		f.movAvgLong = movingaverage.New(f.winLong / f.sampleDuration)
		// Add the current data point back to the long
		// average to start it off.
		f.movAvgLong.Add(data)
	}

	return avg, min, max, avgShort
}

// UpdateReset updates the specified field to the given value
// If the field is an averager window, UpdateReset resets the
// averager
func (f *FlowMovAvg) UpdateReset(whichVal, val int) {
	switch whichVal {
	case WindowLong:
		val = forceMultiple(val, f.sampleDuration)
		f.movAvgLong = movingaverage.New(val / f.sampleDuration)
		f.winLong = val
	case WindowShort:
		val = forceMultiple(val, f.sampleDuration)
		f.movAvgShort = movingaverage.New(val / f.sampleDuration)
		f.winShort = val
	case PercentDiff:
		f.percentDiff = val
	case SampleDuration:
		f.sampleDuration = val
	}
}

func forceMultiple(big, small int) int {
	// Force the input to be a multiple of the sample size
	if big != 0 && small != 0 {
		remainder := big % small
		if remainder != 0 {
			return big + small - remainder
		}
	}

	return big
}
