package keypad

import (
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// Run goroutine for keypad code
func Run(in, out chan interface{}) {
	go func() {
		out <- isdata.KeyLeft
		//for {
		// insert code to read keypad
		// once you have key
		// out <- isdata.KeyLeft
		//}
	}()
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
