package isdata

import (
	"time"

	"github.com/simpleiot/simpleiot/data"
)

// FlowToPulsePeriod calculates the pulse period for a flow (GPH).
// This is used in simulation code.
func FlowToPulsePeriod(flow float64, pulsesPerGal int) time.Duration {
	hoursPerPulse := 1 / (flow * float64(pulsesPerGal))
	usPerPulse := hoursPerPulse * 60 * 60 * 1000 * 1000
	return time.Duration(usPerPulse) * time.Microsecond
}

// PulsesToFlow creates two new Sample structs from pulse data.
// Returns Flow rate is GPH, and amount pumped during this interval in gallons
func PulsesToFlow(tm time.Time, duration time.Duration, pulsesPerGal int, pulses int) (data.Sample, data.Sample) {
	flow := data.Sample{
		Type:     SampleTypeFlowInstantaneous,
		Time:     tm,
		Duration: duration,
	}

	amount := data.Sample{
		Type:  SampleTypeAmount,
		Time:  tm,
		Value: float64(pulses) / float64(pulsesPerGal),
	}

	durationMs := float64(duration.Nanoseconds()) / (1000 * 1000)
	durationHour := durationMs / (1000 * 60.0 * 60.0)

	flow.Value = amount.Value / durationHour

	return flow, amount
}
