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
	menu     *Menu
}

// NewOperatingModeScreen initializes and returns a HomeScreen
func NewOperatingModeScreen(state *isdata.State, config *isdata.Config) *OperatingModeScreen {

	// Find which operating mode to initialize the arrow at
	var selectedIndex int
	switch config.OperatingMode {
	case isdata.ISOperatingModeMonitor:
		selectedIndex = 1
	case isdata.ISOperatingModeMonitorAndShutdown:
		selectedIndex = 0
		// case isdata.ISOperatingModeMonitorAndBatch:
		// selectedIndex = 2
	}

	return &OperatingModeScreen{
		softKeys: NewSoftKeys("back", "setup"),
		state:    state,
		config:   config,
		menu:     NewMenu(true, selectedIndex),
	}
}

// Render updates the home screen, and provides an image
func (s *OperatingModeScreen) Render(img draw.Image) {
	Clear(img)

	s.menu.ResetItems()

	// Find which operating mode is selected
	var monitorSelected, alarmSelected /*, batch*/ bool
	switch s.config.OperatingMode {
	case isdata.ISOperatingModeMonitor:
		monitorSelected = true
	case isdata.ISOperatingModeMonitorAndShutdown:
		alarmSelected = true
		/*case isdata.ISOperatingModeMonitorAndBatch:
		batch = true*/
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
	s.menu.AddItemSelect("Monitor and Shutdown", isdata.UpdateOperatingMode(mode), alarmSelected)
	s.menu.AddItemSelect("Monitor Only", isdata.UpdateOperatingMode(mode), monitorSelected)
	//s.menu.AddItemSelect("Monitor and Batch", isdata.UpdateOperatingMode(mode), batch)

	// render
	s.menu.Render(img)
	Heading(img, "Operating Mode")
	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *OperatingModeScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1Hold: // Back key held -> Home screen
		s.menu.ResetArrowPos() // return arrow to top of screen
		return ScreenIDHome, nil, true
	case isdata.KeySK1Release: // Back
		s.menu.ResetArrowPos() // return arrow to top of screen
		return ScreenIDPrev, nil, true
	case isdata.KeySK2: // Setup
		return ScreenIDOpModeSetup, nil, true
	case isdata.KeyUp, isdata.KeyUpHold, isdata.KeyDown, isdata.KeyDownHold, isdata.KeyRight, isdata.KeyRightHold, isdata.KeyLeft, isdata.KeyLeftHold, isdata.KeyEnter, isdata.KeyEnterHold:
		return s.menu.Key(key)
	}

	return ScreenIDNoChange, nil, true
}
