package isdata

// Define sample types for the system
const (
	SampleTypeFlowWindowAvg  string = "flowWindowAvg"
	SampleTypeAmount                = "amount"
	SampleTypePressure              = "pressure"
	SampleTypePressureVRef          = "pressureVRef"
	SampleTypePressureVSense        = "pressureVSense"
	SampleTypeFaultFlowOff          = "faultFlowOffTarget"
	SampleTypeFaultPresLow          = "faultPressureLow"
	SampleTypeFaultShutdown         = "faultShutdownFailed"
	SampleTypeInputInjector         = "inputInjector"
	SampleTypeInputWaterOn          = "inputWaterOn"
	SampleTypeInputIrrigator        = "inputIrrigator"
	SampleTypeArm                   = "arm"
)

// SampleTypeToDisp converts a sample type code to LCD display string
func SampleTypeToDisp(t string) string {
	switch t {
	case SampleTypeFaultFlowOff:
		return "flow off target"
	case SampleTypeFaultPresLow:
		return "low pressure"
	case SampleTypeFaultShutdown:
		return "shutdown failed"
	default:
		return t
	}
}

// SampleTypeToDispVerbose converts a sample type code to a verbose LCD display string
func SampleTypeToDispVerbose(t string) string {
	switch t {
	case SampleTypeFaultFlowOff:
		return "Shtdwn: flow off target"
	case SampleTypeFaultPresLow:
		return "Shtdwn: pressure low"
	case SampleTypeFaultShutdown:
		return "Failed to shutdown irrigator"
	default:
		return t
	}
}
