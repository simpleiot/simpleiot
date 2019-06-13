package iscontrol

import "github.com/simpleiot/simpleiot/farmation/isdata"

//UpdateFlowStatus updates the flow status state item
func UpdateFlowStatus(state *isdata.State, config *isdata.Config) {
	state.FlowStatus = isdata.FlowStatusArmedOk

	// check if outside lower bound
	if config.ManualLowAlarmGPH > 0 { // if the absolute GPH is set, use it
		if state.FlowRate < config.ManualLowAlarmGPH {
			state.FlowStatus = isdata.FlowStatusOffTarget
		}
	} else { // otherwise, compute a lowerbound in GPH from the percentage
		// target - % * target
		lowBound := config.FlowRateTarget - config.LowWindowPerc/100*config.FlowRateTarget
		if state.FlowRate < lowBound {
			state.FlowStatus = isdata.FlowStatusOffTarget
		}
	}

	// check if outside upper bound
	if config.ManualHighAlarmGPH > 0 {
		if state.FlowRate > config.ManualHighAlarmGPH {
			state.FlowStatus = isdata.FlowStatusOffTarget
		}
	} else {
		// target + % * target
		highBound := config.FlowRateTarget + config.HighWindowPerc/100*config.FlowRateTarget
		if state.FlowRate > highBound {
			state.FlowStatus = isdata.FlowStatusOffTarget
		}
	}

}
