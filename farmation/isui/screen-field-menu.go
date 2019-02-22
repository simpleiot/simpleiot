package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// FieldMenuScreen is used to display status info
type FieldMenuScreen struct {
	softKeys     *SoftKeys
	softKeysEdit *SoftKeys
	state        *isdata.State
	config       *isdata.Config
	arrowPos     int
	menu         Menu
	edit         bool
}

// NewFieldMenuScreen initializes and returns a HomeScreen
func NewFieldMenuScreen(state *isdata.State, config *isdata.Config) *FieldMenuScreen {
	menu := Menu{}
	menu.AddItemScreen("Field One", ScreenIDNoChange)
	menu.AddItemScreen("Field Two", ScreenIDNoChange)
	menu.AddItemScreen("Field Three", ScreenIDNoChange)
	menu.AddItemScreen("Field Four", ScreenIDNoChange)

	return &FieldMenuScreen{
		softKeys:     NewSoftKeys("back", "edit", "import"),
		softKeysEdit: NewSoftKeys("done", "bkspc", "ABC", "cancel"),
		state:        state,
		config:       config,
		menu:         menu,
	}
}

// Render updates the home screen, and provides an image
func (s *FieldMenuScreen) Render(img draw.Image) {
	Clear(img)
	if s.edit {
		// Line(img, 10, 10, 40, 40)
		Heading(img, "Field Name")
		s.softKeysEdit.Render(img, 0, 54)
		DrawTxt(img, "abcdefghijklmnopqrstuvwxyz1", 3, 14, tightpixel15.Font)
		DrawTxt(img, "23456789.", 3, 30, tightpixel15.Font)
	} else {
		Heading(img, "Field Menu")
		s.menu.Render(img)
		s.softKeys.Render(img, 0, 54)
	}
}

// Key processes keypad input to this screen
func (s *FieldMenuScreen) Key(key isdata.Key) (ScreenID, interface{}) {
	if s.edit {
		switch key {
		case isdata.KeySK4:
			s.edit = false
		}
	} else {
		switch key {
		case isdata.KeySK1:
			return ScreenIDMainMenu, nil
		case isdata.KeySK2:
			s.edit = true
		case isdata.KeyUp, isdata.KeyDown, isdata.KeyRight, isdata.KeyLeft, isdata.KeyEnter:
			return s.menu.Key(key)
		}
	}

	return ScreenIDNoChange, nil
}
