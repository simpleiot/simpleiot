package diag

import (
	"errors"
	"fmt"
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

type backupSupplyVoltage struct{}

func (d backupSupplyVoltage) String() string {
	return "backup-voltage"
}

func (d backupSupplyVoltage) Run() (ret error) {
	v, err := isio.ReadVcap()
	if err != nil {
		return err
	}

	vcapNominal := 5.27
	vcapMin := vcapNominal - vcapNominal*0.05
	vcapMax := vcapNominal + vcapNominal*0.05

	if v < vcapMin || v > vcapMax {
		fmt.Println("vcap: ", v)
		return errors.New("Vcap is out of range")
	}

	return
}

func init() {
	Register(pwrGood{})
	Register(backupPwrGood{})
	Register(backupSupplyVoltage{})
}
