package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
)

// DialogScreen is used to display modal dialog messages
type DialogScreen struct {
	softKeys *SoftKeys
}

// NewDialogScreen creates a new dialog screen
func NewDialogScreen() *DialogScreen {
	return &DialogScreen{
		softKeys: NewSoftKeys("OK"),
	}
}

// Render screen
func (s *DialogScreen) Render(img draw.Image, message string) {
	Clear(img)
	Heading(img, "Warning!")

	DrawTxtCentered(img, message, 64, 20,
		tightpixel15.Font)

	s.softKeys.Render(img, 0, 54)
}
