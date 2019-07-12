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
	menu     Menu
}

// NewPumpModeScreen initializes and returns a HomeScreen
func NewPumpModeScreen(state *isdata.State, config *isdata.Config) *PumpModeScreen {

	return &PumpModeScreen{
		softKeys: NewSoftKeys("home"),
		state:    state,
		config:   config,
		menu:     Menu{},
	}
}

// Render updates the home screen, and provides an image
func (s *PumpModeScreen) Render(img draw.Image) {
	Clear(img)

	s.menu.ResetItems()

	// Find which operating mode is selected
	var notset, off, on, inj, acc1, acc2 bool
	switch s.config.UserPumpMode {
	case isdata.UserPumpModeNotSet:
		notset = true
	case isdata.UserPumpModeOff:
		off = true
	case isdata.UserPumpModeOn:
		on = true
	case isdata.UserPumpModeInj:
		inj = true
	case isdata.UserPumpModeAcc1:
		acc1 = true
	case isdata.UserPumpModeAcc2:
		acc2 = true
	}
	var mode int
	switch s.menu.GetArrowPos() {
	case 0:
		mode = int(isdata.UserPumpModeNotSet)
	case 1:
		mode = int(isdata.UserPumpModeOff)
	case 2:
		mode = int(isdata.UserPumpModeOn)
	case 3:
		mode = int(isdata.UserPumpModeInj)
	case 4:
		mode = int(isdata.UserPumpModeAcc1)
	case 5:
		mode = int(isdata.UserPumpModeAcc2)
	}

	// add menu items
	s.menu.AddItemSelect("Not set", isdata.UpdateUserPumpMode(mode), notset)
	s.menu.AddItemSelect("Always Off", isdata.UpdateUserPumpMode(mode), off)
	s.menu.AddItemSelect("Always On", isdata.UpdateUserPumpMode(mode), on)
	s.menu.AddItemSelect("Injector Command", isdata.UpdateUserPumpMode(mode), inj)
	s.menu.AddItemSelect("Lindsay Accessory 1", isdata.UpdateUserPumpMode(mode), acc1)
	s.menu.AddItemSelect("Lindsay Accessory 2", isdata.UpdateUserPumpMode(mode), acc2)

	// render
	s.menu.Render(img)
	Heading(img, "Pump Mode")
	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *PumpModeScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1: // Back
		s.menu.ResetArrowPos() // return arrow to top of screen
		return ScreenIDHome, nil, true
	case isdata.KeyUp, isdata.KeyDown, isdata.KeyRight, isdata.KeyLeft, isdata.KeyEnter:
		return s.menu.Key(key)
	}

	return ScreenIDNoChange, nil, true
}
