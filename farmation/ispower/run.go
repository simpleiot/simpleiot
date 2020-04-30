package ispower

import (
	"log"
	"os/exec"
	"runtime"
	"time"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/farmation/isio"
)

//Run entry point for power management
func Run(in, out chan interface{}) {
	state := isdata.State{}
	ticker := time.NewTicker(time.Second)
	adTicker := time.NewTicker(time.Millisecond * 50)
	averager := data.NewSampleAverager("")

	adcReader := isio.NewAdcReader(isio.AdcVcap)

	if runtime.GOARCH != "arm" {
		ticker.Stop()
		adTicker.Stop()
	}

	lastVcap := 0.0
	powerLossCount := 0
	vStartPowerLoss := 0.0
	poweringOff := false

	for {
		select {
		case m := <-in:
			switch m := m.(type) {
			case isdata.State:
				state = m
			}
		case <-adTicker.C:
			vcap, err := adcReader.Read()
			if err != nil {
				log.Println("Error reading vcap: ", err)
			} else {
				averager.AddSample(data.Sample{Value: vcap / isio.VcapScale})
			}

		case <-ticker.C:
			s := averager.GetAverage()

			if !state.GpioMainAuxPwr {
				if powerLossCount == 0 {
					vStartPowerLoss = s.Value
				}
				log.Println("Power loss count: ", powerLossCount)
				log.Printf("Backup voltage: %.3f, delta: %.3f: \n",
					s.Value, s.Value-lastVcap)
				powerLossCount++
			} else {
				powerLossCount = 0
			}

			lastVcap = s.Value
			averager.ResetAverage()

			if powerLossCount > 3 && !poweringOff {
				log.Printf("after 3sec, backup voltage delta is: %.3f\n", s.Value-vStartPowerLoss)
				if s.Value-vStartPowerLoss < -0.008 {
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
						poweringOff = true
					}

				} else {
					log.Println("After 3 seconds, super cap does not seem to be dropping, start over")
					powerLossCount = 0
				}
			}
		}
	}
}
