package diag

import (
	"errors"
	"fmt"
	"time"

	"github.com/simpleiot/simpleiot/farmation/isio"
)

type adcPressureSense struct{}

func (d adcPressureSense) String() string {
	return "adc-pressure"
}

func (d adcPressureSense) Run() (ret error) {
	ref, sense, err := isio.ReadPressure()
	if err != nil {
		return err
	}

	min := 4.90
	max := 5.10

	if ref < min || ref > max {
		fmt.Println("ref: ", ref)
		return errors.New("pressure ref is out of range")
	}

	if sense < min || sense > max {
		fmt.Println("sense: ", sense)
		return errors.New("pressure sense is out of range, pres ref should be connected to pres sense")
	}

	return nil
}

type adcPanelSense struct{}

func (d adcPanelSense) String() string {
	return "adc-panel-sense"
}

func (d adcPanelSense) Run() (ret error) {
	vPanelSense, err := isio.AdcRead(isio.AdcPanelResistor)
	if err != nil {
		return err
	}

	if vPanelSense < 2.85 || vPanelSense > 2.95 {
		fmt.Println("vPanelSense, expected 2.16, got: ", vPanelSense)
		return errors.New("vPanelSense is out of range")
	}

	return
}

type adcAnalogIn struct{}

func (d adcAnalogIn) String() string {
	return "adc-analog-in"
}

// Set input to 7V and 10mA max
func (d adcAnalogIn) Run() (ret error) {
	isio.SetAnalogInCurrentMode(isio.AdcAnalogIn, false)
	isio.SetAnalogInCurrentMode(isio.AdcAuxIn, false)
	time.Sleep(time.Second)

	v, err := isio.ReadAnalogIn(isio.AdcAnalogIn)
	if err != nil {
		return err
	}

	if v < 6.9 || v > 7.1 {
		fmt.Println("analog in, expected 5V, got: ", v)
		return errors.New("vAnalogIn is out of range")
	}

	isio.SetAnalogInCurrentMode(isio.AdcAnalogIn, true)

	time.Sleep(time.Second * 5)

	i, err := isio.ReadAnalogIn(isio.AdcAnalogIn)
	if err != nil {
		return err
	}

	if i < 0.0093 || i > 0.0108 {
		fmt.Println("analog in, expected 10, got: ", i)
		return errors.New("AnalogIn current is out of range")
	}

	return
}

type adcAuxIn struct{}

func (d adcAuxIn) String() string {
	return "adc-aux-in"
}

// Set input to 7V and 10mA max
func (d adcAuxIn) Run() (ret error) {
	isio.SetAnalogInCurrentMode(isio.AdcAuxIn, false)
	isio.SetAnalogInCurrentMode(isio.AdcAnalogIn, false)
	time.Sleep(time.Second)

	v, err := isio.ReadAnalogIn(isio.AdcAuxIn)
	if err != nil {
		return err
	}

	if v < 6.9 || v > 7.1 {
		fmt.Println("analog in, expected 5V, got: ", v)
		return errors.New("vAnalogIn is out of range")
	}

	isio.SetAnalogInCurrentMode(isio.AdcAuxIn, true)

	time.Sleep(time.Second * 5)

	i, err := isio.ReadAnalogIn(isio.AdcAuxIn)
	if err != nil {
		return err
	}

	if i < 0.0093 || i > 0.0108 {
		fmt.Println("analog in, expected 10, got: ", i)
		return errors.New("AnalogIn current is out of range")
	}

	return
}

func init() {
	//Register(adcPressureSense{})
	//Register(adcPanelSense{})
	//Register(adcAnalogIn{})
	//Register(adcAuxIn{})
}
