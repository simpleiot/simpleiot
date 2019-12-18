package isflow

import (
	"fmt"
	"math"
	"testing"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

func TestAddDataPoint(t *testing.T) {

	config := isdata.Config{
		FlowAvgWindowLong: 30,
		FlowAvgWindow:     3,
		FlowAvgPercDiff:   140,
		SampleDuration:    2,
	}

	fma := NewFlowMovAvg(config.FlowAvgWindowLong, config.FlowAvgWindow, config.FlowAvgPercDiff, config.SampleDuration)

	if fma.winLong%fma.sampleDuration != 0 || fma.winShort%fma.sampleDuration != 0 || fma.sampleDuration == 0 {
		t.Error("Flow windows or sampleDuration not initialized correctly: ", fma.winLong, fma.winShort, fma.sampleDuration)
	}

	var val float64
	for i := 0.1; i < 10000; i = i * 1.5 {
		avg, _, _, avgShort, shortWinUsed := fma.AddDataPoint(i)
		fmt.Println(avg, avgShort, avg == avgShort)
		val = i
		acceptedVal := (avg + avgShort) / 2
		calculatedPercentDiff := int(math.Abs(avgShort-avg) / acceptedVal * 100)
		if calculatedPercentDiff >= config.FlowAvgPercDiff {
			if avg != avgShort || !shortWinUsed {
				t.Error("Diff triggered and didn't replace!!!", calculatedPercentDiff)
			}
		}

	}

	for i := val; i >= 0.1; i = i / 1.5 {
		avg, _, _, avgShort, shortWinUsed := fma.AddDataPoint(i)
		fmt.Println(avg, avgShort, avg == avgShort)
		acceptedVal := (avg + avgShort) / 2
		calculatedPercentDiff := int(math.Abs(avgShort-avg) / acceptedVal * 100)
		if calculatedPercentDiff >= config.FlowAvgPercDiff {
			if avg != avgShort || !shortWinUsed {
				t.Error("Diff triggered and didn't replace!!!", calculatedPercentDiff)
			}
		}
	}

}
