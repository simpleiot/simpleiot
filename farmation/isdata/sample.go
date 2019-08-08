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
	SampleTypeFaultFlowOff             = "faultFlowOffTarget"
	SampleTypeFaultPresLow             = "faultPressureLow"
	SampleTypeFaultShutdown            = "faultShutdownFailed"
	SampleTypeInputInjector            = "inputInjector"
	SampleTypeInputWaterOn             = "inputWaterOn"
	SampleTypeInputIrrigator           = "inputIrrigator"
	SampleTypeArm                      = "arm"
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
