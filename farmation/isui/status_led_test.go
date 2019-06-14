package isui

import (
	"testing"
)

func TestComputeLedState(t *testing.T) {
	/*state := isdata.State{}
	config := isdata.Config{}
	var out chan interface{}
	sl := NewStatusLed(&state, &config, out)

	// Case 1
	sl.UpdateLedState()
	if <-out != isdata.UpdateLedGreen {
		t.Error("SLED first case failed: " + strconv.Itoa(int(sl.LedState)))
	}

	// Case 2 (armed)
	config.Arm = true
	sl.UpdateLedState()
	if sl.LedState != LedGreen {
		t.Error("SLED armed test case failed: " + strconv.Itoa(int(sl.LedState)))
	}

	// Case 3
	// active faults
	state.ActiveFaults = append(state.ActiveFaults, isdata.ISEvent{})
	sl.UpdateLedState()
	if sl.LedState != LedRedBlnk {
		t.Error("SLED active faults case failed: " + strconv.Itoa(int(sl.LedState)))
	}
	state.ActiveFaults = nil
	// flow rate off target
	state.FlowStatus = isdata.FlowStatusOffTarget
	sl.UpdateLedState()
	if sl.LedState != LedRedBlnk {
		t.Error("SLED active faults case failed: " + strconv.Itoa(int(sl.LedState)))
	}
	// both
	state.ActiveFaults = append(state.ActiveFaults, isdata.ISEvent{})
	sl.UpdateLedState()
	if sl.LedState != LedRedBlnk {
		t.Error("SLED active faults case failed: " + strconv.Itoa(int(sl.LedState)))
	}

	// Case 4 (shutdown)
	state.IrrigationShutdown = true
	sl.UpdateLedState()
	if sl.LedState != LedRed {
		t.Error("SLED shutdown case failed: " + strconv.Itoa(int(sl.LedState)))
	}*/
}
