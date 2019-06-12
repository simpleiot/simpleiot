package isio

import (
	"log"
	"runtime"
	"time"

	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/farmation/isui"
)

func setRelay(config *isdata.Config) {
	GpioOut(GpioRelayInjectorEn, config.ManualRelayInj.BoolVal())
	GpioOut(GpioRelayAuxEn, config.ManualRelayAux.BoolVal())
	GpioOut(GpioRelayShutdownEn, config.ManualRelayShutdown.BoolVal())
}

// Run goroutine for IO code
func Run(in, out chan interface{}, configInit isdata.Config, stateInit isdata.State) {
	config := configInit
	state := stateInit

	if runtime.GOARCH == "arm" {
		StatusLightRed(false)
		StatusLightGreen(false)
	}

	gpioReadTicker := time.NewTicker(500 * time.Millisecond) // ticker to read gpio's
	if runtime.GOARCH != "arm" {
		gpioReadTicker.Stop()
	}

	for {
		select {
		case m := <-in:
			switch m := m.(type) {
			case isdata.Config:
				config = m
				if runtime.GOARCH == "arm" {
					setRelay(&config)
				}
			case isdata.State:
				state = m
			case isui.LedState:
				switch m {
				case isui.LedOff:
					GpioOut(GpioStatusRed, false)
					GpioOut(GpioStatusGreen, false)
				case isui.LedRed:
					GpioOut(GpioStatusRed, true)
				case isui.LedGreen:
					GpioOut(GpioStatusGreen, true)
				case isui.LedRedBlnk:
					if GpioRead(GpioStatusRed) {
						GpioOut(GpioStatusRed, false)
					} else {
						GpioOut(GpioStatusRed, true)
					}
				case isui.LedGreenBlnk:
					if GpioRead(GpioStatusGreen) {
						GpioOut(GpioStatusGreen, false)
					} else {
						GpioOut(GpioStatusGreen, true)
					}
				}

			default:
				log.Printf("Isio Mux: unhandled message of type %T: %+v\r\n", m, m)
			}
		case <-gpioReadTicker.C: // if gpio ticker fires
			// if state of gpio changed
			// send new state out on chan

			inj := GpioRead(GpioDigitalInjector)
			if inj != state.GpioDigitalInjector {
				out <- isdata.UpdateGpioDigitalInjector(inj)
			}

			irr := GpioRead(GpioDigitalIrrigator)
			if irr != state.GpioDigitalIrrigator {
				out <- isdata.UpdateGpioDigitalIrrigator(irr)
			}

			water := GpioRead(GpioDigitalWaterOn)
			if water != state.GpioDigitalWaterOn {
				out <- isdata.UpdateGpioDigitalWaterOn(water)
			}

			in := GpioRead(GpioDigitalIn)
			if in != state.GpioDigitalIn {
				out <- isdata.UpdateGpioDigitalIn(in)
			}

		}
	}
}
