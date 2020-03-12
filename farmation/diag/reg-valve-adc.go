package diag

import (
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/simpleiot/simpleiot/farmation/isio"
)

type regValveAnIn struct{}

func (d regValveAnIn) String() string {
	return "reg-valve-analog"
}

// test by connected 10K to REG_VALVE_OUTPUTS
// REG_VALVE_A -> Aux
// REG_VALVE_B -> Analogin
// ADC value with current sens off: 2.92V
// ADC value with current sens on: 0.182V

type regValveAnInConfig struct {
	GpioRegValve string
	Adc          string
}

var regValveAnInTests = []regValveAnInConfig{
	{
		GpioRegValve: isio.GpioRegValve2,
		Adc:          isio.AdcAnalogIn,
	},
	{
		GpioRegValve: isio.GpioRegValve1,
		Adc:          isio.AdcAuxIn,
	},
}

func verify(min, max, v float64) error {
	if v < min || v > max {
		return fmt.Errorf("expected %v, got %v", (min+max)/2, v)
	}

	return nil
}

func (d regValveAnIn) Run() (ret error) {
	for _, t := range regValveAnInTests {
		isio.SetAnalogInCurrentMode(isio.AdcAnalogIn, false)
		isio.SetAnalogInCurrentMode(isio.AdcAuxIn, false)
		isio.GpioOut(isio.GpioRegValve1, false)
		isio.GpioOut(isio.GpioRegValve2, false)
		time.Sleep(time.Second)
		v, err := isio.AdcRead(t.Adc)
		if err != nil {
			errors.Wrap(err, fmt.Sprintf("Error reading: %v", t.Adc))
			return err
		}
		err = verify(-0.01, 0.02, v)
		if err != nil {
			errors.Wrap(err, fmt.Sprintf("%v off", t.Adc))
			return err
		}
		// turn on reg valve output
		isio.GpioOut(t.GpioRegValve, true)
		time.Sleep(time.Millisecond * 100)
		v, err = isio.AdcRead(t.Adc)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Error reading: %v", t.Adc))
		}
		err = verify(2.7, 3, v)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("%v V mode", t.Adc))
		}
		isio.SetAnalogInCurrentMode(t.Adc, true)
		time.Sleep(time.Millisecond * 100)
		v, err = isio.AdcRead(t.Adc)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Error reading: %v", t.Adc))
		}
		err = verify(0.17, 0.195, v)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("%v I mode", t.Adc))
		}
	}

	return nil
}

func init() {
	Register(regValveAnIn{})
}
