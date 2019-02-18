package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// MainMenuScreen is used to display status info
type MainMenuScreen struct {
	softKeys *SoftKeys
	state    *isdata.State
	config   *isdata.Config
	arrowPos int
	menu     Menu
}

// NewMainMenuScreen initializes and returns a HomeScreen
func NewMainMenuScreen(state *isdata.State, config *isdata.Config) *MainMenuScreen {
	softKeys := SoftKeys{}
	softKeys.SetLabel(0, "home")

	menu := Menu{}
	menu.AddItemScreen("Tank Menu", ScreenIDTankMenu1)
	menu.AddItemScreen("Field Menu", ScreenIDFieldMenu1)
	menu.AddItemScreen("Operating Menu", ScreenIDOpMode1)
	menu.AddItemScreen("Totals", ScreenIDTotals)

	return &MainMenuScreen{
		softKeys: &softKeys,
		state:    state,
		config:   config,
		menu:     menu,
	}
}

// Render updates the home screen, and provides an image
func (s *MainMenuScreen) Render(img draw.Image) {
	Clear(img)
	DrawTxt(img, "Main Menu", 37, 2, tightpixel15.Font)
	Rect(img, 33, 1, 51, 10)
	s.menu.Render(img)
	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *MainMenuScreen) Key(key isdata.Key) (ScreenID, interface{}) {
	switch key {
	case isdata.KeySK1:
		return ScreenIDHome, nil
	case isdata.KeyUp, isdata.KeyDown, isdata.KeyRight, isdata.KeyLeft, isdata.KeyEnter:
		return s.menu.Key(key)
	}

	return ScreenIDNoChange, nil
}
