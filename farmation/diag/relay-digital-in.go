package diag

import (
	"errors"
	"log"
	"time"

	"github.com/simpleiot/simpleiot/farmation/isio"
)

type relayDigitalIn struct{}

var errDigitalIn = errors.New("Digital in error")

func (d relayDigitalIn) String() string {
	return "relaydigio"
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
	time.Sleep(100 * time.Millisecond)

	if isio.GpioRead(isio.GpioDigitalInjector) != true {
		log.Println("injector gpio is not active")
		ret = errDigitalIn
	}

	isio.GpioOut(isio.GpioRelayInjectorEn, false)
	time.Sleep(100 * time.Millisecond)

	if isio.GpioRead(isio.GpioDigitalInjector) != false {
		log.Println("injector gpio is active")
		ret = errDigitalIn
	}

	isio.GpioOut(isio.GpioRelayAuxEn, true)
	time.Sleep(100 * time.Millisecond)

	if isio.GpioRead(isio.GpioDigitalIrrigator) != true {
		log.Println("irrigator gpio is not active")
		ret = errDigitalIn
	}

	isio.GpioOut(isio.GpioRelayAuxEn, false)
	time.Sleep(100 * time.Millisecond)

	if isio.GpioRead(isio.GpioDigitalIrrigator) != false {
		log.Println("irrigator gpio is active")
		ret = errDigitalIn
	}

	isio.GpioOut(isio.GpioRelayShutdownEn, true)
	time.Sleep(100 * time.Millisecond)

	if isio.GpioRead(isio.GpioDigitalWaterOn) != true {
		log.Println("water gpio is not active")
		ret = errDigitalIn
	}

	if isio.GpioRead(isio.GpioDigitalIn) != true {
		log.Println("digital in gpio is not active")
		ret = errDigitalIn
	}

	isio.GpioOut(isio.GpioRelayShutdownEn, false)
	time.Sleep(100 * time.Millisecond)

	if isio.GpioRead(isio.GpioDigitalWaterOn) != false {
		log.Println("water gpio is active")
		ret = errDigitalIn
	}

	if isio.GpioRead(isio.GpioDigitalIn) != false {
		log.Println("digital in gpio is active")
		ret = errDigitalIn
	}

	return
}

func init() {
	Register(relayDigitalIn{})
}
