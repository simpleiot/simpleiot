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
	arrowPos int
	menu     Menu
}

// NewOperatingModeScreen initializes and returns a HomeScreen
func NewOperatingModeScreen(state *isdata.State, config *isdata.Config) *OperatingModeScreen {
	menu := Menu{}
	menu.AddItemSelect("Monitor and Shutdown")
	menu.AddItemSelect("Monitor only")
	menu.AddItemSelect("Monitor and Batch")

	return &OperatingModeScreen{
		softKeys: NewSoftKeys("back", "setup"),
		state:    state,
		config:   config,
		menu:     menu,
	}
}

// Render updates the home screen, and provides an image
func (s *OperatingModeScreen) Render(img draw.Image) {
	Clear(img)
	Heading(img, "Operating Mode")
	s.menu.Render(img)
	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *OperatingModeScreen) Key(key isdata.Key) (ScreenID, interface{}) {
	switch key {
	case isdata.KeySK1:
		return ScreenIDMainMenu, nil
	case isdata.KeySK2:
		return ScreenIDOpModeSetup, nil
	case isdata.KeyUp, isdata.KeyDown, isdata.KeyRight, isdata.KeyLeft, isdata.KeyEnter:
		return s.menu.Key(key)
	}

	return ScreenIDNoChange, nil
}
