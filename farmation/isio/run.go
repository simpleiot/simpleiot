package isio

import (
	"log"
	"runtime"
	"time"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

func setRelay(state *isdata.State) {
	GpioOut(GpioRelayInjectorEn, state.GpioRelayInjectorEn)
	GpioOut(GpioRelayAuxEn, state.GpioRelayAuxEn)
	GpioOut(GpioRelayShutdownEn, state.GpioRelayShutdownEn)
}

// Run goroutine for IO code
func Run(in, out chan interface{}, configInit isdata.Config, stateInit isdata.State) {
	config := configInit
	_ = config
	state := stateInit

	if runtime.GOARCH == "arm" {
		StatusLightRed(false)
		StatusLightGreen(false)
	}

	readPanelResistor := func() {
		t, err := GetPanelDefinition()
		_ = t
		if err != nil {
			log.Println("Error reading panel resistor: ", err)
		} else {
			/*
				if state.PanelDefinition != t {
					out <- t
				}
			*/
		}
	}

	gpioReadTicker := time.NewTicker(50 * time.Millisecond) // ticker to read gpio's
	panelSenseTicker := time.NewTicker(10 * time.Second)

	if runtime.GOARCH != "arm" {
		gpioReadTicker.Stop()
		panelSenseTicker.Stop()
	} else {
		readPanelResistor()
	}

	for {
		select {
		case m := <-in:
			switch m := m.(type) {
			case isdata.Config:
				config = m
			case isdata.State:
				state = m
				if runtime.GOARCH == "arm" {
					setRelay(&state)
				}
			case isdata.UpdateLedRed:
				GpioOut(GpioStatusRed, bool(m))
			case isdata.UpdateLedGreen:
				GpioOut(GpioStatusGreen, bool(m))
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
		case <-panelSenseTicker.C:
			readPanelResistor()
		}
	}
}
