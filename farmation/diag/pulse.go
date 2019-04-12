package diag

import (
	"errors"
	"fmt"

	"github.com/simpleiot/simpleiot/farmation/isio"
)

type pulse struct{}

func (d pulse) String() string {
	return "pulse"
}

func (d pulse) Run() error {
	isio.GpioOut(isio.GpioPulseOutput, true)
	fmt.Println("Flow1: ", isio.GpioRead(isio.GpioFlow1Pulse))
	fmt.Println("Flow2: ", isio.GpioRead(isio.GpioFlow2Pulse))
	if isio.GpioRead(isio.GpioFlow1Pulse) != false {
		return errors.New("Flow 1 pulse input should be not active")
	}

	if isio.GpioRead(isio.GpioFlow2Pulse) != false {
		return errors.New("Flow 2 pulse input should be not active")
	}

	isio.GpioOut(isio.GpioPulseOutput, false)
	fmt.Println("Flow1: ", isio.GpioRead(isio.GpioFlow1Pulse))
	fmt.Println("Flow2: ", isio.GpioRead(isio.GpioFlow2Pulse))
	if isio.GpioRead(isio.GpioFlow1Pulse) != true {
		return errors.New("Flow 1 pulse input should be active")
	}

	if isio.GpioRead(isio.GpioFlow2Pulse) != true {
		return errors.New("Flow 2 pulse input should be active")
	}

	return nil
}

func init() {
	Register(pulse{})
}
