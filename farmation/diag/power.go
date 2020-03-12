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

	GetEnter("switch power switch off")

	time.Sleep(10 * time.Millisecond)

	defer GetEnter("switch power switch back on")

	if isio.GpioRead(isio.GpioMainAuxPwr) {
		return errors.New("Failed to detect loss of power")
	}

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
