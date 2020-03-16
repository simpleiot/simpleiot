package diag

import (
	"errors"
	"fmt"
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

type relayFault struct{}

func (d relayFault) String() string {
	return "relay-fault"
}

func blinkGreenLed() {
	isio.GpioOut(isio.GpioStatusGreen, true)
	time.Sleep(200 * time.Millisecond)
	isio.GpioOut(isio.GpioStatusGreen, false)
}

func (d relayFault) Run() error {
	// turn off all relays
	isio.GpioOut(isio.GpioRelayInjectorEn, false)
	isio.GpioOut(isio.GpioRelayShutdownEn, false)
	isio.GpioOut(isio.GpioRelayAuxEn, false)

	defer func() {
		// turn off all relays
		isio.GpioOut(isio.GpioRelayInjectorEn, false)
		isio.GpioOut(isio.GpioRelayShutdownEn, false)
		isio.GpioOut(isio.GpioRelayAuxEn, false)
	}()

	relayFaultGpios := []string{
		isio.GpioRelayInjectorFault,
		isio.GpioRelayAuxFault,
		isio.GpioRelayShutdownFault,
	}

	// make sure all fault signals are inactive to start
	for _, r := range relayFaultGpios {
		if isio.GpioRead(r) {
			return errors.New("relay fault is active for " + r)
		}
	}

	GetEnter("disconnect coils from inj, aux, and shdn relays")

	for _, r := range relayFaultGpios {
		if !isio.GpioRead(r) {
			return fmt.Errorf("open coil fault test failed for %v", r)
		}
	}

	GetEnter("reconnect coils and press enter")

	for _, r := range relayFaultGpios {
		if isio.GpioRead(r) {
			return fmt.Errorf("relay still reading fault after connecting coils %v", r)
		}
	}

	// turn on all relays, and instruct user to short coils
	isio.GpioOut(isio.GpioRelayInjectorEn, true)
	isio.GpioOut(isio.GpioRelayShutdownEn, true)
	isio.GpioOut(isio.GpioRelayAuxEn, true)

	GetEnter("short coils on inj, aux, and shdn relays")

	for _, r := range relayFaultGpios {
		if !isio.GpioRead(r) {
			GetEnter("remove coil shorts")
			return fmt.Errorf("shorted coil fault test failed for %v", r)
		}
	}

	GetEnter("remove coil shorts")

	return nil
}

func init() {
	Register(relayDigitalIn{})
	Register(relayFault{})
}
