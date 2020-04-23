package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// PumpModeScreen is used to display status info
type PumpModeScreen struct {
	softKeys *SoftKeys
	state    *isdata.State
	config   *isdata.Config
	menu     *Menu
}

// NewPumpModeScreen initializes and returns a HomeScreen
func NewPumpModeScreen(state *isdata.State, config *isdata.Config) *PumpModeScreen {

	// Find which pump mode to initialize the arrow at
	var selectedIndex int
	switch config.UserPumpMode {
	case isdata.UserPumpModeOff:
		selectedIndex = 0
	case isdata.UserPumpModeOn:
		selectedIndex = 1
	case isdata.UserPumpModeInj:
		selectedIndex = 2
	case isdata.UserPumpModeAcc1:
		selectedIndex = 3
	case isdata.UserPumpModeAcc2:
		selectedIndex = 4
	}

	return &PumpModeScreen{
		softKeys: NewSoftKeys("back", "test"),
		state:    state,
		config:   config,
		menu:     NewMenu(true, selectedIndex),
	}
}

// Render updates the pump mode screen, and provides an image
func (s *PumpModeScreen) Render(img draw.Image) {
	Clear(img)

	s.menu.ResetItems()

	// Find which operating mode is selected
	var off, on, inj, acc1 bool
	switch s.config.UserPumpMode {
	case isdata.UserPumpModeOff:
		off = true
	case isdata.UserPumpModeOn:
		on = true
	case isdata.UserPumpModeInj:
		inj = true
	case isdata.UserPumpModeAcc1:
		acc1 = true
	}

	// Find correct message for
	var mode int
	switch s.menu.GetArrowPos() {
	case 0:
		mode = int(isdata.UserPumpModeOff)
	case 1:
		mode = int(isdata.UserPumpModeOn)
	case 2:
		mode = int(isdata.UserPumpModeInj)
	case 3:
		mode = int(isdata.UserPumpModeAcc1)
	case 4:
		mode = int(isdata.UserPumpModeAcc2)
	}

	// add menu items
	s.menu.AddItemSelect("Off", isdata.UpdateUserPumpMode(mode), off)
	s.menu.AddItemSelect("On", isdata.UpdateUserPumpMode(mode), on)
	s.menu.AddItemSelect("Injector Command", isdata.UpdateUserPumpMode(mode), inj)

	if s.state.HWVersion == 0 {
		s.menu.AddItemSelect("Vision Serial Acc. 1", isdata.UpdateUserPumpMode(mode), acc1)
	}

	// render
	s.menu.Render(img)
	Heading(img, "Pump Command Source")
	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *PumpModeScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1Release: // Back
		s.menu.ResetArrowPos() // return arrow to top of screen
		return ScreenIDPrev, nil, true
	case isdata.KeySK1Hold: // Home
		s.menu.ResetArrowPos()
		return ScreenIDHome, nil, true
	case isdata.KeySK2: // Test pump
		return ScreenIDPumpTest, nil, true
	case isdata.KeyUp, isdata.KeyUpHold, isdata.KeyDown, isdata.KeyDownHold, isdata.KeyRight, isdata.KeyRightHold, isdata.KeyLeft, isdata.KeyLeftHold, isdata.KeyEnter, isdata.KeyEnterHold:
		return s.menu.Key(key)
	}

	return ScreenIDNoChange, nil, true
}
