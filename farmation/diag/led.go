package diag

import (
	"errors"

	"github.com/simpleiot/simpleiot/farmation/isio"
)

type heartbeat struct{}

func (d heartbeat) Run() error {
	if !GetInput("Is there a heartbeat pattern on the blue led") {
		return errors.New("heartbeat led test failed")
	}
	return nil
}

func (d heartbeat) String() string {
	return "led-heartbeat"
}

type statusGreen struct{}

func (d statusGreen) Run() error {
	isio.GpioOut(isio.GpioStatusGreen, false)
	isio.GpioOut(isio.GpioStatusRed, false)
	if !GetInput("Is the green LED off") {
		return errors.New("green led is not off")
	}
	isio.GpioOut(isio.GpioStatusGreen, true)
	if !GetInput("Is the green LED on") {
		return errors.New("green led is not on")
	}
	isio.GpioOut(isio.GpioStatusGreen, false)
	return nil
}

func (d statusGreen) String() string {
	return "led-status-green"
}

type statusRed struct{}

func (d statusRed) Run() error {
	isio.GpioOut(isio.GpioStatusGreen, false)
	isio.GpioOut(isio.GpioStatusRed, false)
	if !GetInput("Is the red LED off") {
		return errors.New("green led is not off")
	}
	isio.GpioOut(isio.GpioStatusRed, true)
	if !GetInput("Is the red LED on") {
		return errors.New("red led is not on")
	}
	isio.GpioOut(isio.GpioStatusRed, false)
	return nil
}

func (d statusRed) String() string {
	return "led-status-red"
}

func init() {
	Register(heartbeat{})
	Register(statusGreen{})
	Register(statusRed{})
}
