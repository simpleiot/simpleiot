package isdata

import (
	"time"
)

// Flow describes the total and rate over a duration
type Flow struct {
	Duration time.Duration
	Amount   float64
	Rate     float64
}

// PulsesToFlow creates a new Flow struct from pulse data.
// Flow rate is GPH
func PulsesToFlow(duration time.Duration, pulsesPerGal int, pulses int) Flow {
	ret := Flow{
		Duration: duration,
		Amount:   float64(pulses) / float64(pulsesPerGal),
	}

	durationMs := float64(duration.Nanoseconds()) / (1000 * 1000)
	durationHour := durationMs / (1000 * 60.0 * 60.0)

	ret.Rate = ret.Amount / durationHour

	return ret
}
