package isui

import "github.com/simpleiot/simpleiot/data"

// Run goroutine for ui code
func Run(in, out chan interface{}) {
	select {
	case m := <-in:
		switch m := m.(type) {
		case data.Sample:
			// ... todo
			_ = m
		}
	}
}
