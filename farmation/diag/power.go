package diag

import (
	"errors"
	"time"

	"github.com/simpleiot/simpleiot/farmation/isio"
)

type pwrGood struct{}

func (d pwrGood) String() string {
	return "pwr-good"
}

func (d pwrGood) Run() error {
	if !isio.GpioRead(isio.GpioMainAuxPwr) {
		return errors.New("Main Aux power gpio is not active")
	}
	GetInput("switch off 12V power to unit")
	time.Sleep(10 * time.Millisecond)

	if isio.GpioRead(isio.GpioMainAuxPwr) {
		return errors.New("Main Aux power gpio should be inactive when 12V is off")
	}

	GetInput("switch 12V back on")

	return nil
}

type backupPwrGood struct{}

func (d backupPwrGood) String() string {
	return "backup-pwr-good"
}

func (d backupPwrGood) Run() error {
	if !isio.GpioRead(isio.GpioMainAuxPwr) {
		return errors.New("backup power gpio is not active")
	}

	return nil
}

func init() {
	Register(pwrGood{})
	Register(backupPwrGood{})
}
