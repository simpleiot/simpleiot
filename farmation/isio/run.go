package isio

import (
	"github.com/simpleiot/simpleiot/data"
)

// Run goroutine for IO code
func Run(in, out chan interface{}) {
	StatusLightRed(false)
	StatusLightGreen(false)
	for {
		select {
		case m := <-in:
			switch m := m.(type) {
			case data.Sample:
				// ... todo
				_ = m
			}
		}
	}
}
