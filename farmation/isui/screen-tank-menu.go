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
	menu     []string
}

// NewTankMenuScreen initializes and returns a HomeScreen
func NewTankMenuScreen(state *isdata.State, config *isdata.Config) *TankMenuScreen {
	softKeys := SoftKeys{}
	softKeys.SetLabel(0, "back")
	softKeys.SetLabel(1, "full")

	return &TankMenuScreen{
		softKeys: &softKeys,
		state:    state,
		config:   config,
		menu: []string{"Current Volume", "Alert Level",
			"Tank Size", "Tank Alert"},
	}
}

func (s *TankMenuScreen) drawMenu(img draw.Image) {
	for i, entry := range s.menu {
		offset := i * menuSpacingTight
		_ = entry
		DrawTxt(img, entry, 2, 13+offset, tightpixel15.Font)
		Rect(img, 76, 14+offset, 45, menuSpacingTight)
	}
}

// Render updates the home screen, and provides an image
func (s *TankMenuScreen) Render(img draw.Image) {
	Clear(img)
	DrawTxt(img, "Tank Menu", 37, 2, tightpixel15.Font)
	Rect(img, 33, 1, 51, 10)
	s.drawMenu(img)
	Arrow(img, 67, 18+s.arrowPos*menuSpacingTight)

	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *TankMenuScreen) Key(key isdata.Key) (ScreenID, interface{}) {
	switch key {
	case isdata.KeySK1:
		return ScreenMainMenu, nil
	case isdata.KeyUp:
		s.arrowPos--
		if s.arrowPos < 0 {
			s.arrowPos = len(s.menu) - 1
		}
	case isdata.KeyDown:
		s.arrowPos++
		if s.arrowPos >= len(s.menu) {
			s.arrowPos = 0
		}
	case isdata.KeyEnter:
		switch s.arrowPos {
		case 0:
			return ScreenTankMenu1, nil
		case 1:
			return ScreenFieldMenu1, nil
		case 2:
			return ScreenOpMode1, nil
		case 3:
			return ScreenTotals, nil
		}

	}

	return ScreenNoChange, nil
}
