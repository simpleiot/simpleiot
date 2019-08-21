package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// CalibrationScreen calibration screen
type CalibrationScreen struct {
	softKeys *SoftKeys
	state    *isdata.State
	config   *isdata.Config
	menu     Menu
}

// NewCalibrationScreen gives new Totals screen to screen.go
func NewCalibrationScreen(state *isdata.State, config *isdata.Config) *CalibrationScreen {
	isdata.InitState(state) // !!! this comment is from sample screen !!! make sure that ProductStates and FieldStates arrays are large enough
	menu := Menu{}
	menu.AddItemInt("Calibration No.", 123)

	return &CalibrationScreen{
		// update this from sample screen
		softKeys: NewSoftKeys("back", "reset"),
		state:    state,
		config:   config,
		menu:     menu,
	}
}

// Render updates the home screen, and provides an image
func (s *CalibrationScreen) Render(img draw.Image) {
	Clear(img)
	Heading(img, "Calibration")
	s.menu.Render(img)
	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *CalibrationScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1Hold:
		s.menu.ResetArrowPos() // return arrow to top of screen
		return ScreenIDHome, nil, true
	case isdata.KeySK1Release:
		s.menu.ResetArrowPos() // return arrow to top of screen
		return ScreenIDPrev, nil, true
	case isdata.KeyUp, isdata.KeyDown, isdata.KeyRight, isdata.KeyLeft, isdata.KeyEnter:
		return s.menu.Key(key)
	}

	return ScreenIDNoChange, nil, true
}
