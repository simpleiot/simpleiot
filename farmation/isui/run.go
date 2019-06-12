package isui

import (
	"image"
	"log"
	"time"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// Run goroutine for ui code
func Run(in, out chan interface{}, configInit isdata.Config) {
	lcd := image.NewRGBA(image.Rect(0, 0, 128, 64))
	config := configInit
	state := isdata.State{}

	screens := NewScreens(&state, &config)
	dialog := NewDialog()
	widgets := Widgets{}

	widgets.Add(screens)
	widgets.Add(dialog)

	renderScreen := func() {
		widgets.Render(lcd)
		out <- ImageToBlt(0, 0, lcd, false)
		//f, _ := os.OpenFile("lcd.png", os.O_CREATE|os.O_RDWR, 0644)
		//png.Encode(f, lcd)
	}

	renderScreen()

	// initialize the status led struct
	sl := NewStatusLed(&state, &config)
	// Ticker for status LED
	ledTicker := time.NewTicker(350 * time.Millisecond)

	for {
		select {
		case <-ledTicker.C:
			sl.ComputeLedState()
			out <- sl.LedState
		case m := <-in:
			switch m := m.(type) {
			case isdata.State:
				state = m
				renderScreen()
			case isdata.Config:
				config = m
				renderScreen()
			case isdata.Key:
				_, cmd, _ := widgets.Key(m)
				if cmd != nil {
					out <- cmd
				}
				renderScreen()
			default:
				log.Printf("isui mux: unhandled message of type %T: %+v\r\n", m, m)
			}
		}
	}
}
