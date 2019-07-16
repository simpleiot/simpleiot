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
	Heading(img, "Warning")
	font := tightpixel15.Font

	var lengthSoFar, spaceIndex, lastBreak, lineCount int

	for i, char := range message {
		_, charWidth := font.MeasureRune(char)
		lengthSoFar += charWidth + 1

		if string(char) == " " {
			spaceIndex = i
		}

		if (lengthSoFar >= 122 || i >= len(message)-1) && i > 0 { // if message[:i] is the full length of the screen OR at the end of the message

			x := 1
			y := lineCount*font.GetHeight() + 10

			if spaceIndex > font.MeasureString(message)-font.MeasureString(message[lastBreak:]) { // divide by spaces
				DrawTxt(img, message[lastBreak:spaceIndex+1], x, y, font)
				lastBreak = spaceIndex + 1
			} else { // if no spaces encountered
				DrawTxt(img, message[lastBreak:i+1], x, y, font)
				lastBreak = i - 1
			}

			// start a new line
			lengthSoFar = 0
			lineCount++
		}
	}

	s.softKeys.Render(img, 0, 54)
}
