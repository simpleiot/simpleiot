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
}

// NewMainMenuScreen initializes and returns a HomeScreen
func NewMainMenuScreen(state *isdata.State, config *isdata.Config) *MainMenuScreen {
	softKeys := SoftKeys{}
	softKeys.SetLabel(0, "home")

	return &MainMenuScreen{
		softKeys: &softKeys,
		state:    state,
		config:   config,
	}
}

func (s *MainMenuScreen) drawArrow(img draw.Image) {
	offset := s.arrowPos * 11
	Line(img, 34, 17+offset, 40, 17+offset)
	Line(img, 40, 17+offset, 38, 15+offset)
	Line(img, 40, 17+offset, 38, 19+offset)
}

func (s *MainMenuScreen) drawMenu(img draw.Image) {
}

// Render updates the home screen, and provides an image
func (s *MainMenuScreen) Render(img draw.Image) {
	Clear(img)
	DrawTxt(img, "Main Menu", 37, 2, tightpixel15.Font)
	DrawTxt(img, "Tank Menu", 43, 13, tightpixel15.Font)
	DrawTxt(img, "Field Menu", 43, 25, tightpixel15.Font)
	DrawTxt(img, "Operating Menu", 43, 37, tightpixel15.Font)
	DrawTxt(img, "Totals", 43, 48, tightpixel15.Font)
	s.drawArrow(img)

	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *MainMenuScreen) Key(key isdata.Key) (ScreenID, interface{}) {
	switch key {
	case isdata.KeySK1:
		return ScreenHome, nil
	case isdata.KeyUp:
		s.arrowPos--
		if s.arrowPos < 0 {
			s.arrowPos = 4
		}
	case isdata.KeyDown:
		s.arrowPos++
		if s.arrowPos > 4 {
			s.arrowPos = 0
		}
	}

	return ScreenNoChange, nil
}
