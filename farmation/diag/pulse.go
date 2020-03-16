package diag

import (
	"errors"
	"fmt"
	"os/exec"
	"time"

	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/conn/gpio/gpioreg"
)

type pulse struct{}

func (d pulse) String() string {
	return "pulse"
}

func (d pulse) Run() error {
	err := exec.Command("rmmod", "gpio_edge_timer").Run()
	if err != nil {
		fmt.Println("Error removing edge timer driver: ", err)
	}
	time.Sleep(100 * time.Millisecond)

	pOut := gpioreg.ByName("PB7")
	pOut.Out(gpio.Low)

	pIn1 := gpioreg.ByName("PB8")
	pIn2 := gpioreg.ByName("PD13")

	err = pIn1.In(gpio.PullNoChange, gpio.NoEdge)

	if err != nil {
		return err
	}

	err = pIn2.In(gpio.PullNoChange, gpio.NoEdge)

	if err != nil {
		return err
	}

	time.Sleep(10 * time.Millisecond)

	if pIn1.Read() != gpio.High {
		return errors.New("Expected pulse input 1 to be low")
	}

	if pIn2.Read() != gpio.High {
		return errors.New("Expected pulse input 2 to be low")
	}

	pOut.Out(gpio.High)
	time.Sleep(10 * time.Millisecond)

	if pIn1.Read() != gpio.Low {
		return errors.New("Expected pulse input 1 to be high")
	}

	if pIn2.Read() != gpio.Low {
		return errors.New("Expected pulse input 2 to be high")
	}

	return nil
}

func init() {
	Register(pulse{})
}
