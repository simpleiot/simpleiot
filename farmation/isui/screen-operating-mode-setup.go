package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// OperatingModeSetupScreen allows used to change operating mode params
type OperatingModeSetupScreen struct {
	softKeys *SoftKeys
	state    *isdata.State
	config   *isdata.Config
	menu     Menu
}

// NewOperatingModeSetupScreen returns a new screen
func NewOperatingModeSetupScreen(state *isdata.State, config *isdata.Config) *OperatingModeSetupScreen {
	menu := Menu{}
	menu.AddItemInt("High Lev Alm", int(config.HighWindowPerc))
	menu.AddItemInt("Low Lev Alm", int(config.LowWindowPerc))
	menu.AddItemInt("Manual High", int(config.ManualHighAlarmGPH))
	menu.AddItemInt("Manual Low", int(config.ManualLowAlarmGPH))
	menu.AddItemInt("Batch Amount", int(config.BatchAmount))
	menu.AddItemInt("Batch Applied", 0)

	return &OperatingModeSetupScreen{
		softKeys: NewSoftKeys("back", "edit"),
		state:    state,
		config:   config,
		menu:     menu,
	}
}

// Render updates the home screen, and provides an image
func (s *OperatingModeSetupScreen) Render(img draw.Image) {
	Clear(img)
	Heading(img, "Operating Mode Setup")
	s.menu.Render(img)
	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *OperatingModeSetupScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1:
		s.menu.ResetArrowPos() // return arrow to top of screen
		return ScreenIDOpMode1, nil, true
	case isdata.KeyUp, isdata.KeyDown, isdata.KeyRight, isdata.KeyLeft, isdata.KeyEnter:
		return s.menu.Key(key)
	}

	return ScreenIDNoChange, nil, true
}
