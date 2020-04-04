package ispower

import (
	"log"
	"os/exec"
	"runtime"
	"time"

	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/farmation/isio"
)

//Run entry point for power management
func Run(in, out chan interface{}) {
	state := isdata.State{}
	ticker := time.NewTicker(time.Second)

	if runtime.GOARCH != "arm" {
		ticker.Stop()
	}

	powerLossCount := 0
	for {
		select {
		case m := <-in:
			switch m := m.(type) {
			case isdata.State:
				state = m
			}
		case <-ticker.C:
			if !state.GpioMainAuxPwr {
				powerLossCount++
			} else {
				powerLossCount = 0
			}

			if powerLossCount > 3 {
				log.Println("Power loss for 3 seconds, shutting down")
				out <- isdata.Shutdown{}

				// turn off backlight to save power
				isio.GpioOut(isio.GpioLcdPwm, false)

				// shutdown system
				err := exec.Command("poweroff").Start()

				if err != nil {
					log.Println("Error executing power off command")
				} else {
					// sleep forever waiting for power off
					select {}
				}
			}
		}
	}
}
