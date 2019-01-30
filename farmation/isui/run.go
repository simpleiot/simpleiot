package isui

import (
	"image"
	"image/png"
	"os"

	"github.com/simpleiot/simpleiot/data"
)

// Run goroutine for ui code
func Run(in, out chan interface{}) {
	/*
		blt, err := GetLcdAssetBlt(0, 0, "splash.bmp")

		if err != nil {
			log.Println("Error getting blt for splash.bmp")
		} else {
			out <- blt
		}

		time.Sleep(5 * time.Second)

		// clear screen
		out <- isdata.LcdBltSolid{
			X: 0,
			Y: 0,
			W: 128,
			H: 64,
			V: false,
		}

		sk, err := GetLcdAssetBlt(0, 54, "sk-lines.bmp")

		if err != nil {
			log.Println("Error getting blt sk-lines.bmp")
		} else {
			out <- sk
		}

		pumpOn, err := GetLcdAssetBlt(116, 21, "pump-on.bmp")

		if err != nil {
			log.Println("Error getting blt pump-on.bmp")
		}

		pumpOff, err := GetLcdAssetBlt(116, 21, "pump-off.bmp")

		if err != nil {
			log.Println("Error getting blt pump-off.bmp")
		}

		go func() {
			for {
				out <- pumpOn
				time.Sleep(4 * time.Second)
				out <- pumpOff
				time.Sleep(4 * time.Second)
			}
		}()

		out <- RenderTxtBlt(10, 10, "Hello world!")

		go func() {
			flow := 0
			for {
				flow++
				out <- RenderTxtBlt(10, 25, "Flow: "+strconv.Itoa(flow))
				if flow > 50 {
					flow = 0
				}
				time.Sleep(time.Second)
			}
		}()
	*/

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
