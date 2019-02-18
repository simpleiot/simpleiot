package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
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
	softKeys := SoftKeys{}
	softKeys.SetLabel(0, "back")
	softKeys.SetLabel(1, "full")

	menu := Menu{}
	menu.AddItemInt("Current Volume", state.CurrentTankVolume)
	menu.AddItemInt("Alert Level", float64(config.TankAlertVolume))
	menu.AddItemInt("Tank Size", float64(config.TankCapacity))
	menu.AddItemOnOff("Alert On/Off", config.TankAlertOn)

	return &TankMenuScreen{
		softKeys: &softKeys,
		state:    state,
		config:   config,
		menu:     menu,
	}
}

// Render updates the home screen, and provides an image
func (s *TankMenuScreen) Render(img draw.Image) {
	Clear(img)
	DrawTxt(img, "Tank Menu", 37, 2, tightpixel15.Font)
	Rect(img, 33, 1, 51, 10)
	s.menu.Render(img)

	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *TankMenuScreen) Key(key isdata.Key) (ScreenID, interface{}) {
	switch key {
	case isdata.KeySK1:
		return ScreenMainMenu, nil
	case isdata.KeyUp, isdata.KeyDown, isdata.KeyRight, isdata.KeyLeft, isdata.KeyEnter:
		return s.menu.Key(key)
	}

	return ScreenNoChange, nil
}
