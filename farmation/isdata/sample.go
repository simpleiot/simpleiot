package isdata

// Define sample types for the system
const (
	SampleTypePressure string = "pressure"
	// FIXME, Max/Min/Avg can now be stored directly in the sample
	SampleTypePressureMax    string = "pressureMax"
	SampleTypePressureMin    string = "pressureMin"
	SampleTypePressureAvg    string = "pressureAvg"
	SampleTypePressureVRef   string = "pressureVRef"
	SampleTypePressureVSense string = "pressureVSense"
)
