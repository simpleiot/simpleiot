package diag

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/simpleiot/simpleiot/farmation/isio"
	"github.com/simpleiot/simpleiot/farmation/isui"
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

func (d relayFault) Run() error {
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

	fmt.Println("disconnect coils from inj, aux, and shdn relays")

	start := time.Now()

	relayFaultIndex := 0

	for time.Now().Sub(start) < time.Minute && relayFaultIndex < len(relayFaultGpios) {
		r := relayFaultGpios[relayFaultIndex]
		if isio.GpioRead(r) {
			fmt.Println("detected fault for: ", r)
			isui.BlinkGreenLed()
			relayFaultIndex++
		}
		time.Sleep(10 * time.Millisecond)
	}

	if relayFaultIndex < len(relayFaultGpios) {
		return errors.New("open coil fault test failed for: " + relayFaultGpios[relayFaultIndex])
	}

	// make sure all fault signals are inactive again
	time.Sleep(10 * time.Second)
	for _, r := range relayFaultGpios {
		if isio.GpioRead(r) {
			return errors.New("relay fault is active for " + r)
		}
	}

	// turn on all relays, and instruct user to short coils
	isio.GpioOut(isio.GpioRelayInjectorEn, true)
	isio.GpioOut(isio.GpioRelayShutdownEn, true)
	isio.GpioOut(isio.GpioRelayAuxEn, true)

	fmt.Println("short coils on inj, aux, and shdn relays")

	start = time.Now()

	relayFaultIndex = 0

	for time.Now().Sub(start) < time.Minute && relayFaultIndex < len(relayFaultGpios) {
		r := relayFaultGpios[relayFaultIndex]
		if isio.GpioRead(r) {
			fmt.Println("detected short coil for: ", r)
			isui.BlinkGreenLed()
			relayFaultIndex++
		}
		time.Sleep(10 * time.Millisecond)
	}

	if relayFaultIndex < len(relayFaultGpios) {
		return errors.New("shorted coil fault test failed for: " + relayFaultGpios[relayFaultIndex])
	}

	// turn off all relays
	isio.GpioOut(isio.GpioRelayInjectorEn, false)
	isio.GpioOut(isio.GpioRelayShutdownEn, false)
	isio.GpioOut(isio.GpioRelayAuxEn, false)

	return nil
}

func init() {
	Register(relayDigitalIn{})
	Register(relayFault{})
}
