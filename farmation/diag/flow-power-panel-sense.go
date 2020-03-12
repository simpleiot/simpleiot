package diag

import (
	"time"

	"github.com/pkg/errors"

	"github.com/simpleiot/simpleiot/farmation/isio"
)

type flowPowerPanelSense struct{}

func (d flowPowerPanelSense) String() string {
	return "flow-power-panel-sense"
}

func (d flowPowerPanelSense) Run() error {
	isio.GpioOut(isio.GpioRelayInjectorEn, false)
	defer isio.GpioOut(isio.GpioRelayInjectorEn, false)
	time.Sleep(100 * time.Millisecond)
	vPanelSense, err := isio.AdcRead(isio.AdcPanelResistor)
	if err != nil {
		return err
	}
	err = verify(1.3, 1.55, vPanelSense)
	if err != nil {
		return errors.Wrap(err, "Error reading FLOW_1_12V0")
	}

	isio.GpioOut(isio.GpioRelayInjectorEn, true)
	time.Sleep(100 * time.Millisecond)
	vPanelSense, err = isio.AdcRead(isio.AdcPanelResistor)
	if err != nil {
		return err
	}
	err = verify(1.3, 1.55, vPanelSense)
	if err != nil {
		return errors.Wrap(err, "Error reading FLOW_2_12V0")
	}

	return nil
}

func init() {
	Register(flowPowerPanelSense{})
}
