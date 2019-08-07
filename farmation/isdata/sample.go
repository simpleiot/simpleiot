package isdata

// Define sample types for the system
const (
	SampleTypePulses            string = "pulses"
	SampleTypeFlowInstantaneous        = "flowInstantaneous"
	SampleTypeFlowWindowAvg            = "flowWindowAvg"
	SampleTypeAmount                   = "amount"
	SampleTypePressure                 = "pressure"
	SampleTypePressureVRef             = "pressureVRef"
	SampleTypePressureVSense           = "pressureVSense"
	SampleTypeFault                    = "fault"
)

// Define sample sub-types for the system
const (
	SampleSubTypeFaultFlow     string = "flowOffTarget"
	SampleSubTypeFaultPres            = "lowPressure"
	SampleSubTypeFaultShutdown        = "shutdownFailed"
)
