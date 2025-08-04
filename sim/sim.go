package sim

// Sim represents a simulation.
type Sim struct {
	currentValue float64
	step         float64
	max          float64
	min          float64
	up           bool
}

// NewSim creates a new simulation
func NewSim(start, step, minVal, maxVal float64) Sim {
	return Sim{
		currentValue: start,
		step:         step,
		min:          minVal,
		max:          maxVal,
	}
}

// Sim runs a simulation step
func (s *Sim) Sim() float64 {
	if s.up {
		s.currentValue += s.step
		if s.currentValue > s.max {
			s.currentValue = s.max
			s.up = false
		}
	} else {
		s.currentValue -= s.step
		if s.currentValue < s.min {
			s.currentValue = s.min
			s.up = true
		}
	}

	return s.currentValue
}
