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
	menu     []string
}

// NewFieldMenuScreen initializes and returns a HomeScreen
func NewFieldMenuScreen(state *isdata.State, config *isdata.Config) *FieldMenuScreen {
	softKeys := SoftKeys{}
	softKeys.SetLabel(0, "back")
	softKeys.SetLabel(1, "edit")
	softKeys.SetLabel(2, "import")

	return &FieldMenuScreen{
		softKeys: &softKeys,
		state:    state,
		config:   config,
		menu: []string{"Field One", "Field Two",
			"Field Three", "Field Four"},
	}
}

func (s *FieldMenuScreen) drawMenu(img draw.Image) {
	for i, entry := range s.menu {
		DrawTxt(img, entry, 43, 13+i*menuSpacingTight, tightpixel15.Font)
	}
}

// Render updates the home screen, and provides an image
func (s *FieldMenuScreen) Render(img draw.Image) {
	Clear(img)
	DrawTxt(img, "Field Menu", 37, 2, tightpixel15.Font)
	Rect(img, 33, 1, 51, 10)
	s.drawMenu(img)
	Arrow(img, 34, 17+s.arrowPos*menuSpacingTight)

	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *FieldMenuScreen) Key(key isdata.Key) (ScreenID, interface{}) {
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
		case 1:
		case 2:
		case 3:
		}
	}

	return ScreenNoChange, nil
}
