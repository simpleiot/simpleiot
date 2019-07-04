package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// FaultsActiveScreen is used to display active faults
type FaultsActiveScreen struct {
	softKeys *SoftKeys
	state    *isdata.State
	config   *isdata.Config
}

// NewFaultsActiveScreen initializes and returns a screen
func NewFaultsActiveScreen(state *isdata.State, config *isdata.Config) *FaultsActiveScreen {

	return &FaultsActiveScreen{
		softKeys: NewSoftKeys("home", "clear", "histry"),
		state:    state,
		config:   config,
	}
}

// Render updates the home screen, and provides an image
func (s *FaultsActiveScreen) Render(img draw.Image) {
	Clear(img)

	font := tightpixel15.Font
	yPos := 14
	spacing := font.GetHeight() + 2

	if s.state.FaultsActive.Irrigator {
		DrawTxtCentered(img, "IRRIGATOR DIDNT FILL UP", 64, yPos, font)
		yPos += spacing
	}

	Heading(img, "Active Faults")
	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *FaultsActiveScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1: // Home
		return ScreenIDHome, nil, true
	case isdata.KeySK2: // Clear
		return ScreenIDNoChange, isdata.UpdateFaultActiveClearAll{}, true
	case isdata.KeySK3: // History
		return ScreenIDFaultsHistory, nil, true
	}

	return ScreenIDNoChange, nil, true
}
