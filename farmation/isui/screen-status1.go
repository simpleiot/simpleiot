package isui

import (
	"image/draw"
	"strconv"

	"github.com/simpleiot/simpleiot/farmation/fonts/agencyfbbold40"
	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// StatusScreen1 is used to display status info
type StatusScreen1 struct {
	menu   *Menu
	state  *isdata.State
	config *isdata.Config
}

// NewStatusScreen1 initializes and returns a HomeScreen
func NewStatusScreen1(state *isdata.State, config *isdata.Config) *StatusScreen1 {
	menu := Menu{}
	menu.SetLabel(0, "home")
	menu.SetLabel(1, "mode")
	menu.SetLabel(2, "pump")

	return &StatusScreen1{
		menu:   &menu,
		state:  state,
		config: config,
	}
}

// Render updates the home screen, and provides an image
func (s *StatusScreen1) Render(img draw.Image) {
	Clear(img)
	DrawTxt(img, strconv.Itoa(int(s.state.BatchApplied)), 11, 7, agencyfbbold40.Font)
	DrawTxt(img, "GALLONS", 11, 38, tightpixel15.Font)

	s.menu.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *StatusScreen1) Key(key isdata.Key) (ScreenID, interface{}) {
	switch key {
	case isdata.KeyLeft, isdata.KeySK1:
		return ScreenHome, nil
	case isdata.KeyRight:
		return ScreenStatus2, nil
	}

	return ScreenNoChange, nil
}
