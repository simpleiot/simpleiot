package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// StatusScreen3 is used to display status info
type StatusScreen3 struct {
	menu   *Menu
	state  *isdata.State
	config *isdata.Config
}

// NewStatusScreen3 initializes and returns a HomeScreen
func NewStatusScreen3(state *isdata.State, config *isdata.Config) *StatusScreen3 {
	menu := Menu{}
	menu.SetLabel(0, "home")
	menu.SetLabel(1, "mode")
	menu.SetLabel(2, "pump")

	return &StatusScreen3{
		menu:   &menu,
		state:  state,
		config: config,
	}
}

// Render updates the home screen, and provides an image
func (s *StatusScreen3) Render(img draw.Image) {
	Clear(img)
	DrawTxt(img, "Status Screen 3", 11, 38, tightpixel15.Font)

	s.menu.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *StatusScreen3) Key(key isdata.Key) (ScreenID, interface{}) {
	switch key {
	case isdata.KeySK1:
		return ScreenHome, nil
	case isdata.KeyLeft:
		return ScreenStatus2, nil
	case isdata.KeyRight:
		return ScreenHome, nil
	}

	return ScreenNoChange, nil
}
