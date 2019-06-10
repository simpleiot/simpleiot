package iscontrol

import "github.com/simpleiot/simpleiot/farmation/isdata"

//IsFlowGreater23 blah
func IsFlowGreater23(state isdata.State) bool {
	return state.FlowRate > 23
}
