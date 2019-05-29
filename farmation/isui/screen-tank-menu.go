package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// TankMenuScreen is used to display status info
type TankMenuScreen struct {
	softKeys *SoftKeys
	state    *isdata.State
	config   *isdata.Config
	arrowPos int
	menu     Menu
}

// NewTankMenuScreen initializes and returns a HomeScreen
func NewTankMenuScreen(state *isdata.State, config *isdata.Config) *TankMenuScreen {
	menu := Menu{}
	menu.AddItemInt("Current Volume", int(state.CurrentTankVolume))
	menu.AddItemInt("Alert Level", int(config.TankAlertVolume))
	menu.AddItemInt("Tank Size", int(config.TankCapacity))
	menu.AddItemOnOff("Alert On/Off", config.TankAlertOn,
		isdata.UpdateTankAlertEnable(!config.TankAlertOn))

	return &TankMenuScreen{
		softKeys: NewSoftKeys("back", "full"),
		state:    state,
		config:   config,
		menu:     menu,
	}
}

// Render updates the home screen, and provides an image
func (s *TankMenuScreen) Render(img draw.Image) {
	Clear(img)
	Heading(img, "Tank Menu")
	s.menu.Render(img)
	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *TankMenuScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1:
		s.menu.ResetArrowPos() // return arrow to top of screen
		return ScreenIDMainMenu, nil, true
	case isdata.KeyUp, isdata.KeyDown, isdata.KeyRight, isdata.KeyLeft, isdata.KeyEnter:
		return s.menu.Key(key)
	}

	return ScreenIDNoChange, nil, true
}
