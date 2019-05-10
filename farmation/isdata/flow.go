package isdata

import (
	"time"
)

// Flow describes the total and rate over a duration
type Flow struct {
	Time     time.Time
	Duration time.Duration
	Amount   float64
	Rate     float64
	RateAvg  float64
	Pulses   int
}

// PulsesToFlow creates a new Flow struct from pulse data.
// Flow rate is GPH
func PulsesToFlow(tm time.Time, duration time.Duration, pulsesPerGal int, pulses int) Flow {
	ret := Flow{
		Time:     tm,
		Duration: duration,
		Amount:   float64(pulses) / float64(pulsesPerGal),
		Pulses:   pulses,
	}

	durationMs := float64(duration.Nanoseconds()) / (1000 * 1000)
	durationHour := durationMs / (1000 * 60.0 * 60.0)

	ret.Rate = ret.Amount / durationHour

	return ret
}
