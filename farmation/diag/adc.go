package diag

import (
	"errors"
	"fmt"
	"time"

	"github.com/simpleiot/simpleiot/farmation/isio"
)

type adcPanelSense struct{}

func (d adcPanelSense) String() string {
	return "adcPanelSense"
}

func (d adcPanelSense) Run() (ret error) {
	vPanelSense, err := isio.AdcRead(isio.AdcPanelResistor)
	if err != nil {
		return err
	}

	if vPanelSense < 2.15 || vPanelSense > 2.17 {
		fmt.Println("vPanelSense, expected 2.16, got: ", vPanelSense)
		return errors.New("vPanelSense is out of range")
	}

	return
}

type adcAnalogIn struct{}

func (d adcAnalogIn) String() string {
	return "adcAnalogIn"
}

// Set input to 7V and 10mA max
func (d adcAnalogIn) Run() (ret error) {
	isio.SetAnalogInMode(isio.AdcAnalogIn, false)
	time.Sleep(time.Second)

	v, err := isio.ReadAnalogIn(isio.AdcAnalogIn)
	if err != nil {
		return err
	}

	fmt.Println("CLIFF: read volts: ", v)

	if v < 6.9 || v > 7.1 {
		fmt.Println("analog in, expected 5V, got: ", v)
		return errors.New("vAnalogIn is out of range")
	}

	isio.SetAnalogInMode(isio.AdcAnalogIn, true)

	time.Sleep(time.Second * 5)

	i, err := isio.ReadAnalogIn(isio.AdcAnalogIn)
	if err != nil {
		return err
	}

	fmt.Println("CLIFF: read current: ", i)

	if i < 0.0095 || i > 0.0105 {
		fmt.Println("analog in, expected 7mA, got: ", i)
		return errors.New("AnalogIn current is out of range")
	}

	return
}

type adcAuxIn struct{}

func (d adcAuxIn) String() string {
	return "adcAuxIn"
}

// Set input to 7V and 10mA max
func (d adcAuxIn) Run() (ret error) {
	isio.SetAnalogInMode(isio.AdcAuxIn, false)
	time.Sleep(time.Second)

	v, err := isio.ReadAnalogIn(isio.AdcAuxIn)
	if err != nil {
		return err
	}

	fmt.Println("CLIFF: read volts: ", v)

	if v < 6.9 || v > 7.1 {
		fmt.Println("analog in, expected 5V, got: ", v)
		return errors.New("vAnalogIn is out of range")
	}

	isio.SetAnalogInMode(isio.AdcAuxIn, true)

	time.Sleep(time.Second * 5)

	i, err := isio.ReadAnalogIn(isio.AdcAuxIn)
	if err != nil {
		return err
	}

	fmt.Println("CLIFF: read current: ", i)

	if i < 0.0095 || i > 0.0105 {
		fmt.Println("analog in, expected 7mA, got: ", i)
		return errors.New("AnalogIn current is out of range")
	}

	return
}

func init() {
	Register(adcPanelSense{})
	Register(adcAnalogIn{})
	Register(adcAuxIn{})
}
