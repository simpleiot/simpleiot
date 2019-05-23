package isui

import (
	"fmt"
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
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
var alphaUpperLine1 = "ABCDEFGHIJKLM"
var alphaUpperLine2 = "NOPQRSTUVWXYZ"
var numLine = "1234567890 ."

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
		// FIXME add if numbers here
		ret.lines[0] = numLine
	}

	return &ret
}

// Render the widget
func (ic *InputChars) Render(img draw.Image) {

	margin := 31                         // left margin size: x-postion
	line1Y, line2Y, line3Y := 16, 29, 42 // y-postitions of input char lines
	cursorOffset := 9                    // cursor offset from input char lines

	//Input Characters
	DrawTxt(img, ic.lines[0], margin, line1Y, tightpixel15.Font)
	DrawTxt(img, ic.lines[1], margin, line2Y, tightpixel15.Font)
	DrawTxt(img, ic.lines[2], margin, line3Y, tightpixel15.Font)

	xStartPos := tightpixel15.Font.MeasureString(ic.lines[ic.line][:ic.index])
	_, widthChar := tightpixel15.Font.MeasureRune(rune(ic.lines[ic.line][ic.index]))
	switch ic.line {
	case 0: //first line
		Line(img, margin+xStartPos+widthChar/2-2, line1Y+cursorOffset, margin+xStartPos+widthChar/2+2, line1Y+cursorOffset)
	case 1: //second line
		Line(img, margin+xStartPos+widthChar/2-2, line2Y+cursorOffset, margin+xStartPos+widthChar/2+2, line2Y+cursorOffset)
	case 2: //third line
		Line(img, margin+xStartPos+widthChar/2-2, line3Y+cursorOffset, margin+xStartPos+widthChar/2+2, line3Y+cursorOffset)
	}
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
	case isdata.KeyRight:
		currentInputChar = ic.Right()
	case isdata.KeyLeft:
		currentInputChar = ic.Left()
	case isdata.KeyUp:
		currentInputChar = ic.Up()
	case isdata.KeyDown:
		currentInputChar = ic.Down()
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
			if ic.lines[line][index] == c || ic.lines[line][index] == c-32 || ic.lines[line][index] == c+32 {
				ic.line, ic.index = line, index
			} /*else {
				ic.line, ic.index = len(ic.lines)-1, len(ic.lines[ic.line])-1
			}*/
		}
	}
	fmt.Println("c: ", c, "inputchar: ", ic.lines[ic.line][ic.index])

}
