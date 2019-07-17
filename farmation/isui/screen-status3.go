package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// StatusScreen3 is used to display status info
type StatusScreen3 struct {
	softKeys *SoftKeys
	icons    *Icons
	state    *isdata.State
	config   *isdata.Config
}

// NewStatusScreen3 initializes and returns a HomeScreen
func NewStatusScreen3(state *isdata.State, config *isdata.Config) *StatusScreen3 {
	return &StatusScreen3{
		softKeys: NewSoftKeys("home", "mode", "pump"),
		icons:    NewIcons(true, false, false),
		state:    state,
		config:   config,
	}
}

// Render updates the home screen, and provides an image
func (s *StatusScreen3) Render(img draw.Image) {
	Clear(img)
	DrawTxt(img, "Status Screen 3", 11, 38, tightpixel15.Font)

	s.softKeys.Render(img, 0, 54)

	// icons
	s.icons.SetPage("page indicator", 3)
	s.icons.Render(img)
}

// Key processes keypad input to this screen
func (s *StatusScreen3) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1:
		return ScreenIDHome, nil, true
	case isdata.KeyLeft:
		return ScreenIDStatus2, nil, true
	case isdata.KeyRight:
		return ScreenIDHome, nil, true
	}

	return ScreenIDNoChange, nil, true
}
