package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// PumpTestScreen is used to display status info
type PumpTestScreen struct {
	softKeys *SoftKeys
	state    *isdata.State
	config   *isdata.Config
	menu     Menu
}

// NewPumpTestScreen initializes and returns a HomeScreen
func NewPumpTestScreen(state *isdata.State, config *isdata.Config) *PumpTestScreen {

	return &PumpTestScreen{
		softKeys: NewSoftKeys("back", "on", "off"),
		state:    state,
		config:   config,
		menu:     Menu{},
	}
}

// Render updates the pump test screen, and provides an image
func (s *PumpTestScreen) Render(img draw.Image) {
	Clear(img)

	var isAutoMode string
	if s.config.ManualRelayInj == isdata.RelayControlStateAuto {
		isAutoMode = "- auto control mode -"
	} else {
		isAutoMode = "- manual control mode -"
	}

	DrawTxt(img, "Injector Pump Relay is "+BoolToString(s.state.GpioRelayInjectorEn), 8, 19, tightpixel15.Font)
	DrawTxtCentered(img, isAutoMode, 64, 35, tightpixel15.Font)

	Heading(img, "Test Pump")
	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *PumpTestScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1: // Back
		return ScreenIDPrev, isdata.UpdateManualRelayInj(isdata.RelayControlStateAuto), true
	case isdata.KeySK2: // On (pump)
		return ScreenIDNoChange, isdata.UpdateManualRelayInj(isdata.RelayControlStateOn), true
	case isdata.KeySK3: // Off (pump)
		return ScreenIDNoChange, isdata.UpdateManualRelayInj(isdata.RelayControlStateOff), true
	}

	return ScreenIDNoChange, nil, true
}
