package diag

import (
	"errors"
	"log"

	"github.com/simpleiot/simpleiot/farmation/isio"
)

type relayDigitalIn struct{}

var errDigitalIn = errors.New("Digital in error")

func (d relayDigitalIn) String() string {
	return "Digital Input, Relay Driver diags"
}

func (d relayDigitalIn) Run() (ret error) {
	// first, set all relays off and verify digital inputs
	// are high
	isio.GpioOut(isio.GpioRelayInjectorEn, false)
	isio.GpioOut(isio.GpioRelayShutdownEn, false)
	isio.GpioOut(isio.GpioRelayAuxEn, false)

	if isio.GpioRead(isio.GpioDigitalInjector) != false {
		log.Println("injector gpio is not inactive")
		ret = errDigitalIn
	}

	if isio.GpioRead(isio.GpioDigitalIrrigator) != false {
		log.Println("irrigator gpio is not inactive")
		ret = errDigitalIn
	}

	if isio.GpioRead(isio.GpioDigitalWaterOn) != false {
		log.Println("water gpio is not inactive")
		ret = errDigitalIn
	}

	if isio.GpioRead(isio.GpioDigitalIn) != false {
		log.Println("digital in gpio is not inactive")
		ret = errDigitalIn
	}

	// activate one relay driver at a time and verify digital
	// inputs
	isio.GpioOut(isio.GpioRelayInjectorEn, true)

	if isio.GpioRead(isio.GpioDigitalInjector) != true {
		log.Println("injector gpio is not active")
		ret = errDigitalIn
	}

	return
}

func init() {
	Register(relayDigitalIn{})
}
