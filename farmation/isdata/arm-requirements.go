package isdata

// AllArmReqMet is true if all arming requirements are met
func AllArmReqMet(config *Config, state *State) bool {
	for _, v := range ArmReqMet(config, state) {
		if !v {
			return false
		}
	}
	return true
}

// ArmReqMet returns an array of booleans indicating if respective arm requirements are met
func ArmReqMet(config *Config, state *State) [5]bool {
	var ret [5]bool
	ret[0] = state.InputInjector != InputStateOff
	ret[1] = state.InputWaterOn != InputStateOff
	ret[2] = state.InputIrrigator != InputStateOff
	ret[3] = state.FlowRate > 5
	ret[4] = int(state.PressureMin) >= config.PressureStartupLow
	return ret
}
