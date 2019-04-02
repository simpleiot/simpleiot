package diag

import (
	"errors"
	"fmt"
	"time"

	"github.com/simpleiot/simpleiot/farmation/isio"
)

type armSwitch struct{}

func (d armSwitch) Run() error {
	fmt.Println("press arm switch")
	start := time.Now()
	for {
		if time.Since(start) > time.Second*10 {
			return errors.New("timeout waiting for user to press blue switch")
		}

		if isio.GpioRead(isio.GpioArm) {
			break
		}
	}

	return nil
}

func (d armSwitch) String() string {
	return "arm-switch"
}

func init() {
	Register(armSwitch{})
}
