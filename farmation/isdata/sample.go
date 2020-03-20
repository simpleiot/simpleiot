package isdata

// Define sample types for the system
// NOTE: If any "Fault" sample types are added,
// isdb.ReadFaultHist() must be updated with new type
const (
	SampleTypeFlowWindowAvg   string = "flowWindowAvg"
	SampleTypeAmount                 = "amount"
	SampleTypePressure               = "pressure"
	SampleTypePressureVRef           = "pressureVRef"
	SampleTypePressureVSense         = "pressureVSense"
	SampleTypeFaultFlowOff           = "faultFlowOffTarget"
	SampleTypeFaultPresLow           = "faultPressureLow"
	SampleTypeFaultPresHigh          = "faultPressureHigh"
	SampleTypeFaultShutdown          = "faultShutdownFailed"
	SampleTypeFaultNtFlowOff         = "faultNotificationFlowOffTarget"
	SampleTypeFaultNtPresLow         = "faultNotificationPressureLow"
	SampleTypeFaultNtPresHigh        = "faultNotificationPressureHigh"
	SampleTypeInputInjector          = "inputInjector"
	SampleTypeInputWaterOn           = "inputWaterOn"
	SampleTypeInputIrrigator         = "inputIrrigator"
	SampleTypeMainAuxPwr             = "mainAuxPwr"
	SampleTypeArm                    = "arm"
)

// SampleTypeToDisp converts a sample type code to LCD display string
func SampleTypeToDisp(t string) string {
	switch t {
	case SampleTypeFaultFlowOff:
		return "flow off target"
	case SampleTypeFaultPresLow:
		return "low pressure"
	case SampleTypeFaultPresHigh:
		return "high pressure"
	case SampleTypeFaultShutdown:
		return "shutdown failed"
	case SampleTypeFaultNtFlowOff:
		return "notify flow off"
	case SampleTypeFaultNtPresLow:
		return "notify low pres "
	case SampleTypeFaultNtPresHigh:
		return "notify high pres"
	default:
		return t
	}
}

// SampleTypeToDispVerbose converts a sample type code to a verbose LCD display string
func SampleTypeToDispVerbose(t string) string {
	switch t {
	case SampleTypeFaultFlowOff:
		return "Alarm: flow off target"
	case SampleTypeFaultPresLow:
		return "Alarm: pressure low"
	case SampleTypeFaultPresHigh:
		return "Alarm: pressure high"
	case SampleTypeFaultShutdown:
		return "Failed to shutdown irrigator"
	case SampleTypeFaultNtFlowOff:
		return "Notification: flow off"
	case SampleTypeFaultNtPresLow:
		return "Notification: pres low"
	case SampleTypeFaultNtPresHigh:
		return "Notification: pres high"
	default:
		return t
	}
}
