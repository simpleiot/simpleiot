package isui

import (
	"image"
	"image/png"
	"os"

	"github.com/simpleiot/simpleiot/data"
)

// Run goroutine for ui code
func Run(in, out chan interface{}) {
	lcd := image.NewRGBA(image.Rect(0, 0, 128, 64))

	screenHome := NewHomeScreen()
	screenHome.Render(lcd)
	out <- ImageToBlt(0, 0, lcd, false)

	f, _ := os.OpenFile("out.png", os.O_CREATE|os.O_RDWR, 0644)
	png.Encode(f, lcd)

	select {
	case m := <-in:
		switch m := m.(type) {
		case data.Sample:
			// ... todo
			_ = m
		}
	}
}
