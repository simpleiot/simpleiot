package isui

import (
	"time"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// Run goroutine for ui code
func Run(in, out chan interface{}) {

	go func() {
		v := 0
		for {
			time.Sleep(time.Millisecond * 500)
			out <- isdata.MakeBltBlock(10, 10, 20, 20, v)
			if v == 0 {
				v = 1
			} else {
				v = 0
			}
		}
	}()

	go func() {
		x := 0
		y := 0
		v := true

		for {
			time.Sleep(time.Millisecond * 500)
			out <- isdata.LcdPixel{X: x, Y: y, V: v}
			x++
			if x >= 128 {
				x = 0
				y++
				if y >= 32 {
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
