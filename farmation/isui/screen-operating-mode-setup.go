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
	arrowPos int
	menu     Menu
}

// NewOperatingModeSetupScreen returns a new screen
func NewOperatingModeSetupScreen(state *isdata.State, config *isdata.Config) *OperatingModeSetupScreen {
	menu := Menu{}
	menu.AddItemInt("High Level Alarm", float64(config.HighWindowPerc))
	menu.AddItemInt("Low Level Alarm", float64(config.LowWindowPerc))
	menu.AddItemInt("Manual High (GPH)", float64(config.ManualHighAlarmGPH))
	menu.AddItemInt("Manual Low (GPH)", float64(config.ManualLowAlarmGPH))
	menu.AddItemInt("Batch Amount", float64(config.BatchAmount))
	menu.AddItemInt("Batch Applied Amount", 0)

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
func (s *OperatingModeSetupScreen) Key(key isdata.Key) (ScreenID, interface{}) {
	switch key {
	case isdata.KeySK1:
		return ScreenIDMainMenu, nil
	case isdata.KeyUp, isdata.KeyDown, isdata.KeyRight, isdata.KeyLeft, isdata.KeyEnter:
		return s.menu.Key(key)
	}

	return ScreenIDNoChange, nil
}
