package isui

import (
	"time"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// Run goroutine for ui code
func Run(in, out chan interface{}) {

	go func() {
		v := false
		for {
			time.Sleep(time.Millisecond * 500)
			out <- isdata.LcdBltSolid{
				X: 10,
				Y: 10,
				W: 20,
				H: 20,
				V: v}
			v = !v
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
