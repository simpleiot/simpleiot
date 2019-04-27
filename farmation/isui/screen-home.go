package isui

import (
	"image/draw"
	"strconv"

	"github.com/simpleiot/simpleiot/farmation/fonts/agencyfbbold20"
	"github.com/simpleiot/simpleiot/farmation/fonts/agencyfbbold40"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// HomeScreen is used to render the home screen
type HomeScreen struct {
	softKeys *SoftKeys
	state    *isdata.State
	config   *isdata.Config
}

// NewHomeScreen initializes and returns a HomeScreen
func NewHomeScreen(state *isdata.State, config *isdata.Config) *HomeScreen {
	return &HomeScreen{
		softKeys: NewSoftKeys("menu", "mode", "pump"),
		state:    state,
		config:   config,
	}
}

// Render updates the home screen, and provides an image
func (s *HomeScreen) Render(img draw.Image) {
	Clear(img)
	rateS := strconv.FormatFloat(s.state.FlowRate, 'f', 2, 64)
	DrawTxt(img, rateS, 4, 12, agencyfbbold40.Font)
	DrawTxt(img, "963", 67, 11, agencyfbbold20.Font)
	DrawTxt(img, "963", 67, 29, agencyfbbold20.Font)

	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *HomeScreen) Key(key isdata.Key) (ScreenID, interface{}) {
	switch key {
	case isdata.KeyRight:
		return ScreenIDStatus1, nil
	case isdata.KeyLeft:
		return ScreenIDStatus3, nil
	case isdata.KeySK1:
		return ScreenIDMainMenu, nil
	}

	return ScreenIDNoChange, nil
}
