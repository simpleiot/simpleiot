package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// MainMenuScreen is used to display status info
type MainMenuScreen struct {
	softKeys *SoftKeys
	state    *isdata.State
	config   *isdata.Config
	menu     Menu
}

// NewMainMenuScreen initializes and returns a HomeScreen
func NewMainMenuScreen(state *isdata.State, config *isdata.Config) *MainMenuScreen {
	menu := Menu{}
	menu.AddItemScreen("Tank Menu", ScreenIDTankMenu1)
	menu.AddItemScreen("Totals", ScreenIDTotals)
	menu.AddItemScreen("Field Menu", ScreenIDFieldMenu1)
	menu.AddItemScreen("Product Menu", ScreenIDProductMenu1)
	menu.AddItemScreen("Operating Menu", ScreenIDOpMode1)
	menu.AddItemScreen("Diag/Config", ScreenIDDiagConfig)
	//menu.AddItemScreen("Calibration", ScreenIDCalibration)

	return &MainMenuScreen{
		softKeys: NewSoftKeys("home"),
		state:    state,
		config:   config,
		menu:     menu,
	}
}

// Render updates the home screen, and provides an image
func (s *MainMenuScreen) Render(img draw.Image) {
	Clear(img)
	Heading(img, "Main Menu")
	s.menu.Render(img)
	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *MainMenuScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1:
		return ScreenIDHome, nil, true
	case isdata.KeyUp, isdata.KeyUpHold, isdata.KeyDown, isdata.KeyDownHold, isdata.KeyRight, isdata.KeyRightHold, isdata.KeyLeft, isdata.KeyLeftHold, isdata.KeyEnter, isdata.KeyEnterHold:
		return s.menu.Key(key)
	}

	return ScreenIDNoChange, nil, true
}
