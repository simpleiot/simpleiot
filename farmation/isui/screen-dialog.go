package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// DialogScreen is used to display modal dialog messages
type DialogScreen struct {
	softKeys *SoftKeys
	dialog   *isdata.Dialog
}

// NewDialogScreen creates a new dialog screen
func NewDialogScreen(dialog *isdata.Dialog) *DialogScreen {
	return &DialogScreen{
		softKeys: NewSoftKeys("OK"),
		dialog:   dialog,
	}
}

// Render screen
func (s *DialogScreen) Render(img draw.Image) {
	Clear(img)
	Heading(img, "Warning!")

	if s.dialog != nil {
		DrawTxtCentered(img, s.dialog.Message, 64, 20,
			tightpixel15.Font)
	}

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
