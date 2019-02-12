package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// MainMenuScreen is used to display status info
type MainMenuScreen struct {
	menu     *Menu
	state    *isdata.State
	config   *isdata.Config
	arrowPos int
}

// NewMainMenuScreen initializes and returns a HomeScreen
func NewMainMenuScreen(state *isdata.State, config *isdata.Config) *MainMenuScreen {
	menu := Menu{}
	menu.SetLabel(0, "home")
	menu.SetLabel(1, "mode")
	menu.SetLabel(2, "pump")

	return &MainMenuScreen{
		menu:   &menu,
		state:  state,
		config: config,
	}
}

// Render updates the home screen, and provides an image
func (s *MainMenuScreen) Render(img draw.Image) {
	Clear(img)
	DrawTxt(img, "Main Menu", 37, 2, tightpixel15.Font)
	DrawTxt(img, "Tank Menu", 43, 13, tightpixel15.Font)
	DrawTxt(img, "Field Menu", 43, 25, tightpixel15.Font)
	DrawTxt(img, "Operating Menu", 43, 37, tightpixel15.Font)
	DrawTxt(img, "Totals", 43, 48, tightpixel15.Font)

	s.menu.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *MainMenuScreen) Key(key isdata.Key) (ScreenID, interface{}) {
	switch key {
	case isdata.KeySK1:
		return ScreenHome, nil
	}

	return ScreenNoChange, nil
}
