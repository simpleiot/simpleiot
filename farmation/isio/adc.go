package isio

import (
	"errors"
	"fmt"
	"io/ioutil"
	"runtime"
	"strconv"
	"strings"
)

// Defines for various Adc channels
const (
	AdcAnalogIn         string = "AdcAnalogIn"
	AdcPressureSenseRef        = "AdcPressureSenseRef"
	AdcAuxIn                   = "AdcAuxIn"
	AdcPressureSensePos        = "AdcPressureSensePos"
	AdcVcap                    = "AdcVcap"
	AdcPanelResistor           = "AdcPanelResistor"
)

var adcChan = map[string]int{
	AdcAnalogIn:         1,
	AdcPressureSenseRef: 2,
	AdcAuxIn:            4,
	AdcPressureSensePos: 7,
	AdcVcap:             9,
	AdcPanelResistor:    10,
}

// AdcRead is used to read an analog to digital convertor port
func AdcRead(name string) (float64, error) {
	if runtime.GOARCH != "arm" {
		return 0, errors.New("ADC code only runs on Target")
	}
	channel, ok := adcChan[name]
	if !ok {
		return 0, errors.New("invalid ADC name")
	}

	sysfsFile := "/sys/bus/iio/devices/iio:device0/in_voltage" +
		strconv.Itoa(channel) + "_raw"

	countsB, err := ioutil.ReadFile(sysfsFile)
	if err != nil {
		return 0, err
	}

	countsS := strings.TrimSuffix(string(countsB), "\n")

	counts, err := strconv.Atoi(string(countsS))
	if err != nil {
		return 0, err
	}

	scale := 0.201416015

	v := float64(counts) * scale / 1000

	return v, nil
}

// ReadPanelSenseR returns panel sense resistance value in ohms
func ReadPanelSenseR() (res float64, err error) {
	var v float64
	v, err = AdcRead(AdcPanelResistor)
	if err != nil {
		return
	}
	_ = v

	// TODO, finish converting this to panel types

	return
}

var adcNameToCurrentModeSelGpio = map[string]string{
	AdcAnalogIn: GpioAnalogInSel,
	AdcAuxIn:    GpioAuxInSel,
}

var adcCurrentMode = map[string]bool{
	AdcAnalogIn: true,
	AdcAuxIn:    true,
}

// SetAnalogInMode sets voltage or current loop mode
func SetAnalogInMode(name string, currentLoop bool) error {
	gpio, ok := adcNameToCurrentModeSelGpio[name]
	if !ok {
		return errors.New("Could not find select GPIO for analog in")
	}

	fmt.Println("CLIFF: GPIO: ", gpio)

	GpioOut(gpio, currentLoop)
	adcCurrentMode[name] = currentLoop

	return nil
}

// ReadAnalogIn returns the voltage or current of an analog input
func ReadAnalogIn(name string) (float64, error) {

	vad, err := AdcRead(name)
	if err != nil {
		return 0, err
	}

	vin := 3 * vad

	if adcCurrentMode[name] {
		return vin / 499, nil
	}

	return vin, nil
}
