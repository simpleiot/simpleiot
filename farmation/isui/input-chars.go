package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
)

// InputChars is a widget that allows us to enter text (alpha/num)
type InputChars struct {
	lines [3]string
	line  int
	index int
	caps  bool
}

var alphaLowerLine1 = "abcdefghijklm"
var alphaLowerLine2 = "nopqrstuvwxyz"
var alphaUpperLine1 = "ABCDEFGHIJKL"
var alphaUpperLine2 = "MNOPQRSTUVWXYZ"
var numLine = "1234567890-/. "

// NewInputChars creates a new inputchars widget that allows character selection.
// alpha enables input.
// numbers enables number input.
func NewInputChars(alpha, numbers bool) *InputChars {
	ret := InputChars{}
	if alpha {
		ret.lines[0], ret.lines[1] = alphaLowerLine1, alphaLowerLine2
		if numbers {
			ret.lines[2] = numLine
		}
	} else {
		ret.lines[0] = numLine
	}

	return &ret
}

// Render the widget
func (ic *InputChars) Render(img draw.Image) {
	DrawTxt(img, ic.lines[0], 31, 16, tightpixel15.Font)
	DrawTxt(img, ic.lines[1], 31, 29, tightpixel15.Font)
	DrawTxt(img, ic.lines[2], 31, 42, tightpixel15.Font)
}

// Caps sets to upper case
func (ic *InputChars) Caps(enable bool) {
	if enable {
		ic.lines[0] = alphaUpperLine1
		ic.lines[1] = alphaUpperLine2
	} else {
		ic.lines[0] = alphaLowerLine1
		ic.lines[1] = alphaLowerLine2
	}
}

// Right moves cursor right
func (ic *InputChars) Right() byte {
	ic.index++
	if ic.index >= len(ic.lines[ic.line]) {
		ic.index = 0
		ic.line++
	}

	if ic.line >= len(ic.lines) {
		ic.line = 0
	}

	return ic.GetCurrent()
}

// GetCurrent returns the current character from the input
func (ic *InputChars) GetCurrent() byte {
	return ic.lines[ic.line][ic.index]
}

// IndexTo moves the cursor to the character specified
func (ic *InputChars) IndexTo(c byte) {
	for i := 0; i <= len(ic.lines)-1; i++ {
		for j := 0; j <= len(ic.lines[i]); j++ {
			if ic.lines[i][j] == c {
				ic.line, ic.index = i, j
			}
		}
	}

}
