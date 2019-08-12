package isui

import (
	"image/draw"
	"strconv"

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

	lenFaults := len(s.state.FaultsActive) - 1
	for i := lenFaults; i >= 0; i-- {

		// only diplay three active faults
		if lenFaults-i >= 3 {
			break
		}

		msg := isdata.SampleTypeToDispVerbose(s.state.FaultsActive[i].Type)
		if s.state.FaultsActive[i].Type != isdata.SampleTypeFaultShutdown {
			msg += ", " + strconv.FormatFloat(s.state.FaultsActive[i].Value, 'f', 0, 64)
		}

		DrawTxt(img, msg, 1, yPos, font)
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
