package isio

import (
	"fmt"
	"log"
	"runtime"

	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/conn/gpio/gpioreg"
	"periph.io/x/periph/host"
)

// define gpios
const (
	GpioDigitalInjector  string = "GpioDigitalInjector"
	GpioDigitalIrrigator        = "GpioDigitalIrrigator"
	GpioDigitalWaterOn          = "GpioDigitalWaterOn"
	GpioDigitalIn               = "GpioDigitalIn"

	GpioRelayInjectorEn = "GpioRelayInjectorEn"
	GpioRelayShutdownEn = "GpioRelayShutdownEn"
	GpioRelayAuxEn      = "GpioRelayAuxEn"

	// select between voltage or current loop input
	// high = current loop
	GpioAnalogInSel = "GpioAnalogInSel"
	GpioAuxInSel    = "GpioAuxInSel"
)

type pin struct {
	Port      string
	ActiveLow bool
	Output    bool
	Pin       gpio.PinIO
}

var pins = map[string]*pin{
	GpioDigitalInjector:  &pin{"PC24", true, false, nil},
	GpioDigitalIrrigator: &pin{"PD8", true, false, nil},
	GpioDigitalWaterOn:   &pin{"PD9", true, false, nil},
	GpioDigitalIn:        &pin{"PD24", true, false, nil},

	GpioRelayInjectorEn: &pin{"PC13", true, true, nil},
	GpioRelayShutdownEn: &pin{"PC9", true, true, nil},
	GpioRelayAuxEn:      &pin{"PB0", true, true, nil},

	GpioAnalogInSel: &pin{"PC15", false, false, nil},
	GpioAuxInSel:    &pin{"PD25", false, false, nil},
}

// GpioInit is used to initialize gpios
func GpioInit() {
	// Load periph.io drivers:
	if _, err := host.Init(); err != nil {
		log.Println("Error initializing periph.io host", err)
		return
	}

	if runtime.GOARCH == "arm" {
		for k, v := range pins {
			pins[k].Pin = gpioreg.ByName(v.Port)
			if pins[k].Pin == nil {
				log.Println("Warning, could init Gpio: ", k, v.Port)
			} else if !v.Output {
				pins[k].Pin.In(gpio.PullNoChange, gpio.NoEdge)
			}
		}
	}

}

// GpioOut sets a Gpio value
func GpioOut(name string, value bool) {
	p, ok := pins[name]
	if !ok || p.Pin == nil {
		log.Println("Error setting gpio: ", name)
		return
	}

	fmt.Println("CLIFF: p: ", p)

	p.Pin.Out(gpio.Level(value != p.ActiveLow))
}

// GpioRead reads the current Gpio value
func GpioRead(name string) bool {
	p, ok := pins[name]
	if !ok || p.Pin == nil {
		log.Println("Error reading gpio: ", name)
		return false
	}

	return bool(p.Pin.Read()) != p.ActiveLow
}
