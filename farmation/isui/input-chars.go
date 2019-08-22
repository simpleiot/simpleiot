package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15fixed"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// InputChars is a widget that allows us to enter text (alpha/num)
type InputChars struct {
	lines       [3]string
	line        int
	index       int
	numbersOnly bool
	caps        bool
}

var alphaLowerLine1 = "abcdefghijklm"
var alphaLowerLine2 = "nopqrstuvwxyz"
var alphaUpperLine1 = "ABCDEFGHIJKLM"
var alphaUpperLine2 = "NOPQRSTUVWXYZ"
var numLine = "0123456789 ./"

// NewInputChars creates a new inputchars widget that allows character selection.
// alpha enables input.
// numbers enables number input.
func NewInputChars(alpha, numbers bool) *InputChars {
	ret := InputChars{}
	if alpha {
		ret.lines[0], ret.lines[1] = alphaLowerLine1, alphaLowerLine2 // just alpha input chars
		if numbers {                                                  // alpha and numbers
			ret.lines[2] = numLine
		}
	} else if numbers { // just numbers
		ret.lines[0] = numLine[:10] // slice off space and period
	} else { // one null input char
		ret.lines[0] = "\x00"
	}

	if numbers && !alpha {
		ret.numbersOnly = true
	}

	return &ret
}

// Render the widget
func (ic *InputChars) Render(img draw.Image) {

	margin := 19                         // left margin size: x-postion
	line1Y, line2Y, line3Y := 17, 30, 43 // y-postitions of input char lines

	//Input Characters
	currentChar, caps := rune(ic.lines[ic.line][ic.index]), ic.caps
	if ic.line == 2 || ic.lines[1] == "" { // if on numbers/symbols line
		caps = true // highlight like caps
	}
	DrawTxtHighlight(img, ic.lines[0], currentChar, caps, margin, line1Y, tightpixel15fixed.Font)
	DrawTxtHighlight(img, ic.lines[1], currentChar, caps, margin, line2Y, tightpixel15fixed.Font)
	DrawTxtHighlight(img, ic.lines[2], currentChar, caps, margin, line3Y, tightpixel15fixed.Font)

}

// Caps sets to upper case
func (ic *InputChars) Caps() byte {
	if ic.caps {
		ic.lines[0] = alphaLowerLine1
		ic.lines[1] = alphaLowerLine2
		ic.caps = false
	} else {
		ic.lines[0] = alphaUpperLine1
		ic.lines[1] = alphaUpperLine2
		ic.caps = true
	}

	return ic.GetCurrent()
}

// Key handles key inputs specific to inputChars
func (ic *InputChars) Key(key isdata.Key) byte {
	currentInputChar := "\x00"[0] // null byte
	switch key {
	case isdata.KeySK3: // Caps
		currentInputChar = ic.Caps()
	case isdata.KeyRight, isdata.KeyRightHold:
		currentInputChar = ic.Right()
	case isdata.KeyLeft, isdata.KeyLeftHold:
		currentInputChar = ic.Left()
	case isdata.KeyUp, isdata.KeyUpHold:
		if ic.numbersOnly {
			currentInputChar = ic.Right()
		} else {
			currentInputChar = ic.Up()
		}
	case isdata.KeyDown, isdata.KeyDownHold:
		if ic.numbersOnly {
			currentInputChar = ic.Left()
		} else {
			currentInputChar = ic.Down()
		}
	}

	return currentInputChar
}

// Right moves cursor right
func (ic *InputChars) Right() byte {
	ic.index++
	if ic.index >= len(ic.lines[ic.line]) {
		ic.index = 0
		if ic.line >= len(ic.lines)-1 { // if we're at the end
			ic.line = 0
		} else if len(ic.lines[ic.line+1]) > 0 { // else if next line isn't an empty string
			ic.line++
		} else {
			ic.line = 0
		}
	}

	return ic.GetCurrent()
}

// Left moves cursor left
func (ic *InputChars) Left() byte {
	ic.index--
	if ic.index < 0 {
		ic.line--
		if ic.line < 0 {
			if len(ic.lines[len(ic.lines)-1]) > 0 { // if last line isn't an empty string
				ic.line = len(ic.lines) - 1
			} else if len(ic.lines[len(ic.lines)-2]) > 0 { // else if second to last isn't empty
				ic.line = len(ic.lines) - 2
			} else {
				ic.line = 0
			}
		}
		ic.index = len(ic.lines[ic.line]) - 1
	}

	return ic.GetCurrent()
}

//Up moves cursor up a line
func (ic *InputChars) Up() byte {
	ic.line--
	if ic.line < 0 {
		if len(ic.lines[len(ic.lines)-1]) > 0 {
			ic.line = len(ic.lines) - 1
		} else if len(ic.lines[len(ic.lines)-2]) > 0 {
			ic.line = len(ic.lines) - 2
		} else {
			ic.line = 0
		}
	}

	if ic.index >= len(ic.lines[ic.line]) {
		ic.index = len(ic.lines[ic.line]) - 1
	}

	return ic.GetCurrent()
}

//Down moves cursor down a line
func (ic *InputChars) Down() byte {
	ic.line++
	if ic.line >= len(ic.lines) || len(ic.lines[ic.line]) <= 0 { //if we're past the end or this line is empty
		ic.line = 0
	}
	if ic.index >= len(ic.lines[ic.line]) {
		ic.index = len(ic.lines[ic.line]) - 1
	}

	return ic.GetCurrent()
}

// GetCurrent returns the current character from the input
func (ic *InputChars) GetCurrent() byte {
	return ic.lines[ic.line][ic.index]
}

// IndexTo moves the cursor to the character specified
func (ic *InputChars) IndexTo(c byte) {
	for line := 0; line <= len(ic.lines)-1; line++ {
		for index := 0; index <= len(ic.lines[line])-1; index++ {
			if ic.lines[line][index] == c { // if current char == c
				ic.line, ic.index = line, index
			} else if ic.lines[line][index] == c+32 && c >= 65 && c <= 90 { // if current char == lowercase c and c is a upper case alpha
				if ic.caps == false { // if current char is lowercase
					ic.Caps()
					ic.line, ic.index = line, index
				}
			} else if ic.lines[line][index] == c-32 && c >= 97 && c <= 122 { // if current char == uppercase c and c is a lowercase alpha
				if ic.caps { // if current char is uppercase
					ic.Caps()
					ic.line, ic.index = line, index
				}
			}
		}
	}
}
