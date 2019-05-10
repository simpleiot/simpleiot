package isui

import "image/draw"

// InputChars is a widget that allows us to enter text (alpha/num)
type InputChars struct {
	lines []string
	line  int
	index int
	caps  bool
}

var alphaLowerLine1 = "abcdefghijklm"
var alphaUpperLine1 = "ABCDEFGHIJKLM"

// NewInputChars creates a new inputchars widget that allows character selection.
// alpha enables input.
// numbers enables number input.
func NewInputChars(alpha, numbers bool) *InputChars {
	if alpha {

	}
	/*
		lowerCaseInput := inputChars{
			Lines: []string{"abcdefghijklm",
				"nopqrstuvwxyz",
				"0123456789. ",
			},
		}

		upperCaseInput := inputChars{
			Lines: []string{"ABCDEFGHIJKL",
				"MNOPQRSTUVWXYZ",
				"0123456789. ",
			},
		}
	*/

	return &InputChars{}
}

// Render the widget
func (ic *InputChars) Render(img draw.Image) {
}

// Caps sets to upper case
func (ic *InputChars) Caps(enable bool) {
	if enable {
		ic.lines[0] = alphaUpperLine1
	} else {
		ic.lines[0] = alphaLowerLine1
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
