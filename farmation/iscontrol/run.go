package iscontrol

import (
	"time"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// Run goroutine for ui code
func Run(in, out chan interface{}, configInit isdata.Config, stateInit isdata.State) {
	config := configInit
	state := stateInit
	flowStatusUpdateTicker := time.NewTicker(1000 * time.Millisecond)
	var seconds int // timer to send out flow rate off target + faults

	for {
		select {
		case <-flowStatusUpdateTicker.C:
			flowStatus := GetFlowStatus(&state, &config)
			if state.FlowStatus != flowStatus {
				if flowStatus == isdata.FlowStatusOffTarget {
					seconds++
					if seconds >= config.AlarmRecognizeSec {
						out <- isdata.UpdateFlowStatus(flowStatus)
					}
				} else {
					out <- isdata.UpdateFlowStatus(flowStatus)
					seconds = 0
				}
			}
		case m := <-in:
			switch m := m.(type) {
			case isdata.State:
				state = m
			case isdata.Config:
				config = m
			}
		}

	}
}
