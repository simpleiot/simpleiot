package isflow

import (
	"math"
	"testing"

	movingaverage "github.com/RobinUS2/golang-moving-average"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

func TestAddDataPoint(t *testing.T) {

	config := isdata.Config{
		FlowAvgWindowLong: 30,
		FlowAvgWindow:     300,
		FlowAvgPercDiff:   25,
	}

	fma := NewFlowMovAvg(config.FlowAvgWindowLong, config.FlowAvgWindow, config.FlowAvgPercDiff)

	compareLong := movingaverage.New(config.FlowAvgWindowLong)
	compareShort := movingaverage.New(config.FlowAvgWindow)

	for i := 0.0; i < 100.0; i = i + 0.3 {
		avg, min, max := fma.AddDataPoint(i)

		compareLong.Add(i)
		compareShort.Add(i)
		avgLong := compareLong.Avg()
		avgShort := compareShort.Avg()

		// Compare
		calculatedDiff := int(math.Abs(avgLong-avgShort) / avgLong)
		// If the long average should have been returned
		if calculatedDiff > config.FlowAvgPercDiff/100 {
		}
	}
}
