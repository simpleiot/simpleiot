package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// OperatingModeScreen is used to display status info
type OperatingModeScreen struct {
	softKeys *SoftKeys
	state    *isdata.State
	config   *isdata.Config
	menu     Menu
}

// NewOperatingModeScreen initializes and returns a HomeScreen
func NewOperatingModeScreen(state *isdata.State, config *isdata.Config) *OperatingModeScreen {

	return &OperatingModeScreen{
		softKeys: NewSoftKeys("back", "setup"),
		state:    state,
		config:   config,
		menu:     Menu{},
	}
}

// Render updates the home screen, and provides an image
func (s *OperatingModeScreen) Render(img draw.Image) {
	Clear(img)

	s.menu.ResetItems()

	// Find which operating mode is selected
	var monitor, shtdwn, batch bool
	switch s.config.OperatingMode {
	case isdata.ISOperatingModeMonitor:
		monitor = true
	case isdata.ISOperatingModeMonitorAndShutdown:
		shtdwn = true
	case isdata.ISOperatingModeMonitorAndBatch:
		batch = true
	}
	var mode int
	switch s.menu.GetArrowPos() {
	case 0:
		mode = int(isdata.ISOperatingModeMonitorAndShutdown)
	case 1:
		mode = int(isdata.ISOperatingModeMonitor)
	case 2:
		mode = int(isdata.ISOperatingModeMonitorAndBatch)
	}

	// add menu items
	s.menu.AddItemSelect("Monitor and Shutdown", isdata.UpdateOperatingMode(mode), shtdwn)
	s.menu.AddItemSelect("Monitor only", isdata.UpdateOperatingMode(mode), monitor)
	s.menu.AddItemSelect("Monitor and Batch", isdata.UpdateOperatingMode(mode), batch)

	// render
	s.menu.Render(img)
	Heading(img, "Operating Mode")
	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *OperatingModeScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1:
		s.menu.ResetArrowPos() // return arrow to top of screen
		return ScreenIDMainMenu, nil, true
	case isdata.KeySK2:
		return ScreenIDOpModeSetup, nil, true
	case isdata.KeyUp, isdata.KeyDown, isdata.KeyRight, isdata.KeyLeft, isdata.KeyEnter:
		return s.menu.Key(key)
	}

	return ScreenIDNoChange, nil, true
}
