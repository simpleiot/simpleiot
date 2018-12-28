package isui

import (
	"time"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// Run goroutine for ui code
func Run(in, out chan interface{}) {

	x := 0
	y := 0
	v := true
	go func() {
		for {
			time.Sleep(time.Microsecond * 100)
			out <- isdata.LcdPixel{X: x, Y: y, V: v}
			x++
			if x >= 128 {
				x = 0
				y++
				if y >= 64 {
					y = 0
					v = !v
				}
			}
		}
	}()

	select {
	case m := <-in:
		switch m := m.(type) {
		case data.Sample:
			// ... todo
			_ = m
		}
	}
}
