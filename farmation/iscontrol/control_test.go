package iscontrol

import (
	"testing"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

func TestUpdateFlowStatus(t *testing.T) {
	state := isdata.State{FlowRate: 40}
	config := isdata.Config{
		FlowRateTarget: 40,
		LowWindowPerc:  10,
		HighWindowPerc: 20,
	}

	// Case 1 (flow rate on target)
	UpdateFlowStatus(&state, &config)
	if state.FlowStatus != isdata.FlowStatusArmedOk {
		t.Error("FlOWSTATUS test failed, Case 1")
	}

	// Case 2 (flow rate high, use percentage)
	state.FlowRate = 60
	UpdateFlowStatus(&state, &config)
	if state.FlowStatus != isdata.FlowStatusOffTarget {
		t.Error("FlOWSTATUS test failed, Case 2")
	}

	// Case 3 (flow rate high, use absolute GPH)
	config.ManualHighAlarmGPH = 50
	UpdateFlowStatus(&state, &config)
	if state.FlowStatus != isdata.FlowStatusOffTarget {
		t.Error("FlOWSTATUS test failed, Case 3")
	}

	// Case 4 (flow rate on target)
	state.FlowRate = 41
	UpdateFlowStatus(&state, &config)
	if state.FlowStatus != isdata.FlowStatusArmedOk {
		t.Error("FlOWSTATUS test failed, Case 4")
	}

	// Case 5 (flow rate low, use percentage)
	state.FlowRate = 30
	UpdateFlowStatus(&state, &config)
	if state.FlowStatus != isdata.FlowStatusOffTarget {
		t.Error("FlOWSTATUS test failed, Case 5")
	}

	// Case 6 (flow rate low, use absolute GPH)
	config.ManualLowAlarmGPH = 35
	UpdateFlowStatus(&state, &config)
	if state.FlowStatus != isdata.FlowStatusOffTarget {
		t.Error("FlOWSTATUS test failed, Case 6")
	}

}
