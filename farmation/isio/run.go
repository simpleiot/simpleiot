package isio

import (
	"time"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

func setRelay(config *isdata.Config) {
	GpioOut(GpioRelayInjectorEn, config.ManualRelayInj)
	GpioOut(GpioRelayAuxEn, config.ManualRelayAux)
	GpioOut(GpioRelayInjectorEn, config.ManualRelayInj)

}

// Run goroutine for IO code
func Run(in, out chan interface{}, configInit isdata.Config, stateInit isdata.State) {
	config := configInit
	state := stateInit
	StatusLightRed(false)
	StatusLightGreen(false)
	gpioReadTicker := time.NewTicker(500 * time.Millisecond) // ticker to read gpio's
	for {
		select {
		case m := <-in:
			switch m := m.(type) {
			case isdata.Config:
				config = m
				setRelay(&config)
			case data.Sample:
				// ... todo
				_ = m
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
