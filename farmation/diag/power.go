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

	if isio.GpioRead(isio.GpioBackupPwr) {
		return errors.New("backup power gpio is active")
	}

	GetEnter("switch power switch off")

	time.Sleep(10 * time.Millisecond)

	defer GetEnter("switch power switch back on")

	if isio.GpioRead(isio.GpioMainAuxPwr) {
		return errors.New("Failed to detect loss of power")
	}

	if !isio.GpioRead(isio.GpioBackupPwr) {
		return errors.New("Failed to detect we are on backup power")
	}

	return nil
}

type backupSupplyVoltage struct{}

func (d backupSupplyVoltage) String() string {
	return "backup-voltage"
}

func (d backupSupplyVoltage) Run() error {
	v, err := isio.ReadVcap()
	if err != nil {
		return err
	}

	vcapNominal := 5.27
	vcapMin := vcapNominal - vcapNominal*0.05
	vcapMax := vcapNominal + vcapNominal*0.05

	if v > vcapMax {
		return fmt.Errorf("Vcap is high, expected 5.27, got %v", v)
	}

	if v < vcapMin {
		// check if vcap is rising at 4mV/sec -- see
		// https://trello.com/c/S6ro11Ix for data on how fast the
		// super caps charge
		fmt.Println("checking if caps are charging, please wait ...")
		time.Sleep(10 * time.Second)
		v2, err := isio.ReadVcap()
		if err != nil {
			return err
		}

		minVChange := 8 * 0.00433
		vChange := v2 - v

		if vChange < minVChange {
			return fmt.Errorf("caps are not charging at expected rate, got %v, expected %v", vChange, minVChange)
		}
	}

	// caps are charged and everything looks good
	return nil
}

func init() {
	Register(pwrGood{})
	Register(backupSupplyVoltage{})
}
