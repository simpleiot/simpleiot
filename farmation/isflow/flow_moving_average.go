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
}

// Define values to reference different averagers by
const (
	MovAvgLong int = iota
	MovAvgShort
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
func (f FlowMovAvg) AddDataPoint(data float64) (avg, min, max float64) {
	f.movAvgLong.Add(data)
	f.movAvgShort.Add(data)

	avgLong := f.movAvgLong.Avg()
	avg = f.movAvgShort.Avg()
	min, _ = f.movAvgShort.Min()
	max, _ = f.movAvgShort.Max()

	// If the flow rate calculated from the short-window
	// moving average is inconsistent with the rate from
	// the long-window average, set the output flow rate
	// to the average from the long window
	calculatedDiff := int(math.Abs(avgLong-avg) / avgLong)
	if calculatedDiff > f.percentDiff/100 {
		avg = avgLong
	}

	return avg, min, max
}

// ResetAvg resets the average specified by whichAvg to
// the time window given by win
func (f FlowMovAvg) ResetAvg(whichAvg, win int) {
	switch whichAvg {
	case MovAvgLong:
		f.movAvgLong = movingaverage.New(win)
	case MovAvgShort:
		f.movAvgShort = movingaverage.New(win)
	}
}

// UpdatePercentDiff is used to update this stored field in the
// FlowMovAvg struct if this value changes in the system config
func (f FlowMovAvg) UpdatePercentDiff(newVal int) {
	f.percentDiff = newVal
}
