package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// DialogScreen is used to display modal dialog messages
type DialogScreen struct {
	softKeys *SoftKeys
	state    *isdata.State
	config   *isdata.Config
}

// NewDialogScreen creates a new dialog screen
func NewDialogScreen(state *isdata.State, config *isdata.Config) *DialogScreen {
	return &DialogScreen{
		softKeys: NewSoftKeys("OK"),
		state:    state,
		config:   config,
	}
}

// Render screen
func (s *DialogScreen) Render(img draw.Image) {
	Clear(img)
	Heading(img, "Warning!")

	DrawTxtCentered(img, s.state.Dialog.Message, 64, 20,
		tightpixel15.Font)

	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *DialogScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1:
		return ScreenIDNoChange, isdata.UpdateDialogAck{}, true
	}

	return ScreenIDNoChange, nil, true
}
