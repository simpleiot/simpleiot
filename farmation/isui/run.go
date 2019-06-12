package isui

import (
	"image"
	"log"
	"time"

	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/farmation/isio"
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

	// Ticker for status LED
	ledTicker := time.NewTicker(750 * time.Millisecond)

	for {
		select {
		case <-ledTicker.C:
			isio.StatusLightRed(false)
			isio.StatusLightGreen(false)
			switch {
			case config.Arm:

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
