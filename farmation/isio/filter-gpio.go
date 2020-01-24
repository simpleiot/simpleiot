package isio

import "time"

// FilterGpioSignal is a system to prevent spurious noise on the gpio's from
// clogging the system
type FilterGpioSignal struct {
	out       chan interface{}
	lastReset time.Time
	toggleCnt int
}

// Toggle takes a new value and an old value and decides whether to send out
// the new one on the out channel
func (f *FilterGpioSignal) Toggle(valOld, valNew bool) {

	if time.Since(f.lastReset) > 5*time.Second {
		f.toggleCnt = 0
		f.lastReset = time.Now()
	}

	if valNew == valOld || f.toggleCnt > 3 {
		return
	}

	f.out <- valNew
	f.toggleCnt++
}
