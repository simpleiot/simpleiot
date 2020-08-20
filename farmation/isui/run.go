package isui

import (
	"image"
	"log"
	"time"

	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/farmation/isdb"
)

// Run goroutine for ui code
func Run(in, out chan interface{}, configInit isdata.Config, stateInit isdata.State, db *isdb.IsDb) {
	lcd := image.NewRGBA(image.Rect(0, 0, 128, 64))
	config := configInit
	state := stateInit

	screens := NewScreens(&state, &config, db)
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

	lastKey := time.Now()
	screenSaver := false

	renderTicker := time.NewTicker(500 * time.Millisecond)

	for {
		select {
		case <-renderTicker.C:
			if time.Since(lastKey) > 30*time.Minute && !screenSaver {
				// show screen saver instead
				screens.ScreenSaver(true)
				screenSaver = true
				out <- isdata.SetBacklight(false)
			}
			renderScreen()
		case m := <-in:
			switch m := m.(type) {
			case isdata.State:
				state = m
				renderScreen()
			case isdata.Config:
				config = m
				renderScreen()
			case isdata.Key:
				// Don't clear the screen saver with a soft key 1 press, because
				// it will always be followed by a release, and we don't want that
				// to take action, so it must be "absorbed" by the screen saver.
				if screenSaver && m == isdata.KeySK1 {
					continue
				}
				_, cmd, _ := widgets.Key(m)
				if cmd != nil {
					out <- cmd
				}
				lastKey = time.Now()
				if screenSaver {
					screenSaver = false
					screens.ScreenSaver(false)
				}
				out <- isdata.SetBacklight(true)
				renderScreen()
			default:
				log.Printf("isui mux: unhandled message of type %T: %+v\r\n", m, m)
			}
		}
	}
}
