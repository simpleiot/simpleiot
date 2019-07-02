package isio

import (
	"errors"
	"io/ioutil"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// Defines for various Adc channels
const (
	AdcAnalogIn      string = "AdcAnalogIn"
	AdcPressureRef          = "AdcPressureRef"
	AdcAuxIn                = "AdcAuxIn"
	AdcPressureSense        = "AdcPressureSense"
	AdcVcap                 = "AdcVcap"
	AdcPanelResistor        = "AdcPanelResistor"
)

var adcChan = map[string]int{
	AdcAnalogIn:      1,
	AdcPressureRef:   2,
	AdcAuxIn:         4,
	AdcPressureSense: 7,
	AdcVcap:          9,
	AdcPanelResistor: 10,
}

var adcMutex sync.Mutex

// AdcReadCount is used to read A/D count times and average results
func AdcReadCount(name string, count int) (ret float64, err error) {
	samples := make([]float64, count)
	for i := 0; i < count; i++ {
		samples[i], err = AdcRead(name)
		if err != nil {
			return
		}
	}

	for _, v := range samples {
		ret += v
	}

	ret = ret / float64(count)

	return
}

// AdcRead is used to read an analog to digital convertor port
func AdcRead(name string) (float64, error) {
	adcMutex.Lock()
	defer adcMutex.Unlock()

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

// Below is the max voltage expected for panel, so the idea is you loop through
// starting at the beginning of the array and check if the voltage is less than
var panelDefinitions = []isdata.PanelDefinition{
	{0.459, isdata.PanelTypeInvalid, "Invalid"},
	{0.763, isdata.PanelTypeLindsay, "Lindsay"},
	{1.072, isdata.PanelTypeValleyIconSerial, "Valley Icon Serial"},
	{1.402, isdata.PanelTypeValleyCam, "Valley CAM"},
	{1.720, isdata.PanelTypeRinkySerial, "Rinky Serial"},
	{2.021, isdata.PanelTypeReserved, "Reserved"},
	{2.382, isdata.PanelTypeReserved, "Reserved"},
	{2.770, isdata.PanelTypeStandardPump, "Standard Pump"},
	{3.124, isdata.PanelTypeStandardPivot, "Standard Pivot"},
	{5.000, isdata.PanelTypeInvalid, "Invalid"},
}

func getPanelDefintion(v float64) (def isdata.PanelDefinition) {
	for _, d := range panelDefinitions {
		if v < d.Voltage {
			def = d
			break
		}
	}

	def.Voltage = v
	if def.Description == "" {
		def.Description = "Invalid"
	}

	return
}

// GetPanelDefinition returns panel definition
func GetPanelDefinition() (def isdata.PanelDefinition, err error) {
	var v float64
	v, err = ReadPanelSenseR()
	if err != nil {
		return
	}

	return getPanelDefintion(v), nil
}

// ReadPanelSenseR returns panel sense resistance value in ohms
func ReadPanelSenseR() (res float64, err error) {
	var v float64
	v, err = AdcReadCount(AdcPanelResistor, 10)
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

// SetAnalogInCurrentMode sets voltage or current loop mode
func SetAnalogInCurrentMode(name string, currentLoop bool) error {
	gpio, ok := adcNameToCurrentModeSelGpio[name]
	if !ok {
		return errors.New("Could not find select GPIO for analog in")
	}

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
		return vin / 249.5, nil
	}

	return vin, nil
}

var vcapScale = 0.376257545

// ReadVcap returns the voltage of the supercap supply
func ReadVcap() (v float64, err error) {
	v, err = AdcRead(AdcVcap)
	v = v / vcapScale
	return
}

var pressureScale = 0.62825

// ReadPressure reads the supply and pressure voltage
func ReadPressure() (ref float64, sense float64, err error) {
	ref, err = AdcRead(AdcPressureRef)
	ref = ref / pressureScale
	if err != nil {
		return
	}
	sense, err = AdcRead(AdcPressureSense)
	sense = sense / pressureScale
	if err != nil {
		return
	}
	return
}

// ReadPressureSense reads only the pressure sense voltage
// the idea is you should not have to read the ref very often
func ReadPressureSense() (sense float64, err error) {
	sense, err = AdcRead(AdcPressureSense)
	sense = sense / pressureScale
	if err != nil {
		return
	}
	return
}
