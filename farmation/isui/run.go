package isui

import (
	"log"
	"time"

	"github.com/simpleiot/simpleiot/data"
)

// Run goroutine for ui code
func Run(in, out chan interface{}) {

	bmpTest()

	blt, err := GetLcdAssetBlt(0, 0, "splash.bmp")

	if err != nil {
		log.Println("Error getting blt for splash.bmp")
	} else {
		go func() {
			time.Sleep(5 * time.Second)
			out <- blt
		}()
	}

	select {
	case m := <-in:
		switch m := m.(type) {
		case data.Sample:
			// ... todo
			_ = m
		}
	}
}
