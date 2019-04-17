package diag

import (
	"errors"

	"github.com/simpleiot/simpleiot/farmation/isio"
)

type lcd struct{}

func (d lcd) String() string {
	return "lcd"
}

func (d lcd) Run() error {
	if !GetInput("Does LCD look OK") {
		return errors.New("LCD inspection failed")
	}
	return nil
}

type backlight struct{}

func (d backlight) String() string {
	return "backlight"
}

func (d backlight) Run() error {
	isio.GpioOut(isio.GpioLcdPwm, false)
	if !GetInput("is backlight off") {
		return errors.New("lcd backlight should be off")
	}

	isio.GpioOut(isio.GpioLcdPwm, true)
	if !GetInput("is backlight on") {
		return errors.New("lcd backlight should be on")
	}

	return nil
}

func init() {
	Register(lcd{})
	Register(backlight{})
}
