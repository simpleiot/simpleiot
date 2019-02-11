package isui

import (
	"image"
	"image/png"
	"log"
	"os"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// Run goroutine for ui code
func Run(in, out chan interface{}, configInit *isdata.Config) {
	lcd := image.NewRGBA(image.Rect(0, 0, 128, 64))
	config := configInit
	state := &isdata.State{}

	var currentScreen Screen
	screens := InitScreens(state, config)

	currentScreen = screens[ScreenHome]

	renderScreen := func() {
		currentScreen.Render(lcd)
		out <- ImageToBlt(0, 0, lcd, false)
		f, _ := os.OpenFile("lcd.png", os.O_CREATE|os.O_RDWR, 0644)
		png.Encode(f, lcd)
	}

	renderScreen()

	for {
		select {
		case m := <-in:
			switch m := m.(type) {
			case isdata.State:
				*state = m
				renderScreen()
			case isdata.Config:
				*config = m
				renderScreen()
			case isdata.Key:
				newScreen, cmd := currentScreen.Key(m)
				if cmd != nil {
					out <- cmd
				}
				if newScreen != ScreenNoChange {
					currentScreen = screens[newScreen]
				}
				renderScreen()
			default:
				log.Printf("Mux: unhandled message of type %T: %+v\r\n", m, m)
			}
		}
	}
}
