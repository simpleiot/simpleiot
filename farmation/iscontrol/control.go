package iscontrol

import "github.com/simpleiot/simpleiot/farmation/isdata"

//GetFlowStatus updates the flow status state item
func GetFlowStatus(state *isdata.State, config *isdata.Config) isdata.FlowStatus {
	flowStatus := isdata.FlowStatusArmedOk

	// check if outside lower bound
	if config.ManualLowAlarmGPH > 0 { // if the absolute GPH is set, use it
		if state.FlowRate < config.ManualLowAlarmGPH {
			flowStatus = isdata.FlowStatusOffTarget
		}
	} else { // otherwise, compute a lowerbound in GPH from the percentage
		// target - % * target
		lowBound := config.FlowRateTarget - config.LowWindowPerc/100*config.FlowRateTarget
		if state.FlowRate < lowBound {
			flowStatus = isdata.FlowStatusOffTarget
		}
	}

	// check if outside upper bound
	if config.ManualHighAlarmGPH > 0 {
		if state.FlowRate > config.ManualHighAlarmGPH {
			flowStatus = isdata.FlowStatusOffTarget
		}
	} else {
		// target + % * target
		highBound := config.FlowRateTarget + config.HighWindowPerc/100*config.FlowRateTarget
		if state.FlowRate > highBound {
			flowStatus = isdata.FlowStatusOffTarget
		}
	}

	return flowStatus

}
