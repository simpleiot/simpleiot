package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/fonts/agencyfbbold20"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// FaultsActiveScreen is used to display status info
type FaultsActiveScreen struct {
	softKeys *SoftKeys
	state    *isdata.State
	config   *isdata.Config
}

// NewFaultsActiveScreen initializes and returns a screen
func NewFaultsActiveScreen(state *isdata.State, config *isdata.Config) *FaultsActiveScreen {

	return &FaultsActiveScreen{
		softKeys: NewSoftKeys("back", "clear", "histry"),
		state:    state,
		config:   config,
	}
}

// Render updates the home screen, and provides an image
func (s *FaultsActiveScreen) Render(img draw.Image) {
	Clear(img)

	switch {
	case s.state.FaultsActive.Irrigator:
		DrawTxtCentered(img, "Irrigator failed to fill up", 64, 32, agencyfbbold20.Font)
	}

	Heading(img, "Active Faults")
	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *FaultsActiveScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1: // Back
		return ScreenIDPrev, nil, true
	case isdata.KeySK2: // Clear
		return ScreenIDNoChange, isdata.UpdateFaultIrrigator(false), true
	case isdata.KeySK3: // History
	}

	return ScreenIDNoChange, nil, true
}
