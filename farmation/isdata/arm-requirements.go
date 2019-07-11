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
	ret[0] = state.GpioDigitalWaterOn
	ret[1] = state.GpioDigitalInjector
	ret[2] = state.GpioDigitalIrrigator
	ret[3] = state.FlowRate > 0
	ret[4] = int(state.PressureMin) >= config.PressureStartupLow
	return ret
}
