package isio

import (
	"errors"
	"io/ioutil"
	"os"
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
	{Voltage: 0.459, Type: isdata.PanelTypeInvalid},
	{Voltage: 0.763, Type: isdata.PanelTypeLindsay},
	{Voltage: 1.072, Type: isdata.PanelTypeValleyIconSerial},
	{Voltage: 1.402, Type: isdata.PanelTypeValleyCam},
	{Voltage: 1.720, Type: isdata.PanelTypeRinkySerial},
	{Voltage: 2.021, Type: isdata.PanelTypeReserved},
	{Voltage: 2.382, Type: isdata.PanelTypeReserved},
	{Voltage: 2.770, Type: isdata.PanelTypeStandardPump},
	{Voltage: 3.124, Type: isdata.PanelTypeStandardPivot},
	{Voltage: 5.000, Type: isdata.PanelTypeInvalid},
}

func getPanelDefintion(v float64) (def isdata.PanelDefinition) {
	for _, d := range panelDefinitions {
		if v < d.Voltage {
			def = d
			break
		}
	}

	def.Voltage = v

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
	res, err = AdcReadCount(AdcPanelResistor, 10)
	if err != nil {
		res = 0
		return
	}

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

// VcapScale can be used to scale A/D when reading backup cap voltage
var VcapScale = 0.376257545

// ReadVcap returns the voltage of the supercap supply
func ReadVcap() (v float64, err error) {
	v, err = AdcRead(AdcVcap)
	v = v / VcapScale
	return
}

var pressureScale = 0.62825

// ReadPressure reads the supply and pressure voltage
func ReadPressure() (ref float64, sense float64, err error) {
	if runtime.GOARCH != "arm" {
		return
	}

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
	if runtime.GOARCH != "arm" {
		return
	}

	sense, err = AdcRead(AdcPressureSense)
	sense = sense / pressureScale
	if err != nil {
		return
	}
	return
}

// AdcReader is used to keep A/D file open for consecutive reads
type AdcReader struct {
	adc string
	f   *os.File
}

// NewAdcReader creates a new adc reader
func NewAdcReader(adc string) *AdcReader {
	return &AdcReader{
		adc: adc,
	}
}

// Read returns the voltage from the A/D
func (adc *AdcReader) Read() (float64, error) {
	if adc.f == nil {
		// need to open adc first

		if runtime.GOARCH != "arm" {
			return 0, errors.New("ADC code only runs on Target")
		}

		channel, ok := adcChan[adc.adc]

		if !ok {
			return 0, errors.New("invalid ADC name")
		}

		sysfsFile := "/sys/bus/iio/devices/iio:device0/in_voltage" +
			strconv.Itoa(channel) + "_raw"

		var err error
		adc.f, err = os.Open(sysfsFile)

		if err != nil {
			return 0, err
		}
	}

	_, err := adc.f.Seek(0, 0)

	if err != nil {
		return 0, err
	}

	b := make([]byte, 50)
	c, err := adc.f.Read(b)
	b = b[:c]

	if err != nil {
		return 0, err
	}

	countsS := strings.TrimSuffix(string(b), "\n")

	counts, err := strconv.Atoi(countsS)
	if err != nil {
		return 0, err
	}

	scale := 0.201416015

	v := float64(counts) * scale / 1000

	return v, nil
}
