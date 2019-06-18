package iscontrol

import "github.com/simpleiot/simpleiot/farmation/isdata"

//GetFlowStatus updates the flow status state item
func GetFlowStatus(state *isdata.State, config *isdata.Config) isdata.FlowStatus {
	highBound, lowBound := config.CalculateFlowWindow()

	flowStatus := isdata.FlowStatusArmedOk
	if state.FlowRate > highBound || state.FlowRate < lowBound {
		flowStatus = isdata.FlowStatusOffTarget
	}

	return flowStatus

}
