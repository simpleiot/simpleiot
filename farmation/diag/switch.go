package diag

import (
	"errors"
	"fmt"
	"time"
)

type blueSwitch struct{}

func (d blueSwitch) Run() error {
	fmt.Println("press blue switch")
	start := time.Now()
	for {
		if time.Since(start) > time.Second*10 {
			return errors.New("timeout waiting for user to press blue switch")
		}

		/*
			if gio.GetSwitch() {
				break
			}
		*/
	}

	return nil
}

func (d blueSwitch) String() string {
	return "switch"
}

func init() {
	/*
		var blueSwitchDiag blueSwitch
		Register(blueSwitchDiag)
	*/
}
