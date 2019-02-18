package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// FieldMenuScreen is used to display status info
type FieldMenuScreen struct {
	softKeys *SoftKeys
	state    *isdata.State
	config   *isdata.Config
	arrowPos int
	menu     Menu
}

// NewFieldMenuScreen initializes and returns a HomeScreen
func NewFieldMenuScreen(state *isdata.State, config *isdata.Config) *FieldMenuScreen {
	softKeys := SoftKeys{}
	softKeys.SetLabel(0, "back")
	softKeys.SetLabel(1, "edit")
	softKeys.SetLabel(2, "import")

	menu := Menu{}
	menu.AddItemScreen("Field One", ScreenIDNoChange)
	menu.AddItemScreen("Field Two", ScreenIDNoChange)
	menu.AddItemScreen("Field Three", ScreenIDNoChange)
	menu.AddItemScreen("Field Four", ScreenIDNoChange)

	return &FieldMenuScreen{
		softKeys: &softKeys,
		state:    state,
		config:   config,
		menu:     menu,
	}
}

// Render updates the home screen, and provides an image
func (s *FieldMenuScreen) Render(img draw.Image) {
	Clear(img)
	DrawTxt(img, "Field Menu", 37, 2, tightpixel15.Font)
	Rect(img, 33, 1, 51, 10)
	s.menu.Render(img)
	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *FieldMenuScreen) Key(key isdata.Key) (ScreenID, interface{}) {
	switch key {
	case isdata.KeySK1:
		return ScreenIDMainMenu, nil
	case isdata.KeyUp, isdata.KeyDown, isdata.KeyRight, isdata.KeyLeft, isdata.KeyEnter:
		return s.menu.Key(key)
	}

	return ScreenIDNoChange, nil
}
