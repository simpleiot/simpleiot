package isui

import (
	"fmt"
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

	if font.MeasureString(message) <= 126 {
		DrawTxtCentered(img, message, 64, 20, font)
	} else {
		var spaceIndex int

		for i, char := range message {
			lengthSoFar := font.MeasureString(message[:i])

			if string(char) == " " {
				spaceIndex = i
			}
			if lengthSoFar >= 126 && i > 0 { // if message[:i] is the full length of the screen
				if spaceIndex == 0 { // if no spaces encountered
					DrawTxtCentered(img, message[:i-1], 64, 20, font)
					DrawTxtCentered(img, message[i-1:], 64, 29, font)
					break
				} else { // divide by spaces
					message2 := message[spaceIndex+1:]
					if len(message2) <= 126 { // if message will fit in two lines
						DrawTxtCentered(img, message[:spaceIndex+1], 64, 20, font)
						DrawTxtCentered(img, message2, 64, 29, font)
						break
					} else { // three lines
						fmt.Println("Dialog: 3 lines")
						var spaceIndex2 int
						for i, char := range message2 {
							lengthSoFar := font.MeasureString(message2[:i])

							if string(char) == " " {
								spaceIndex2 = i
							}
							if lengthSoFar >= 126 && i > 0 { // if message2[:i] is the full length of the screen
								DrawTxtCentered(img, message[:spaceIndex+1], 64, 20, font)
								DrawTxtCentered(img, message2[:spaceIndex2+1], 64, 29, font)
								DrawTxtCentered(img, message2[spaceIndex2+1:], 64, 29, font)
								break
							}
						}
					}
				}
			}
		}
	}

	s.softKeys.Render(img, 0, 54)
}
