package isui

import (
	"fmt"
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

var alphaLowerLine1 = "abcdefghijklmn"
var alphaLowerLine2 = "opqrstuvwxyz"
var alphaUpperLine1 = "ABCDEFGHIJKLM"
var alphaUpperLine2 = "NOPQRSTUVWXYZ"
var numLine = "1234567890. "

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
func (ic *InputChars) Render(img draw.Image, textEntry bool) {

	margin := 31 //right

	//Input Characters
	DrawTxt(img, ic.lines[0], 31, 16, tightpixel15.Font)
	DrawTxt(img, ic.lines[1], 31, 29, tightpixel15.Font)
	DrawTxt(img, ic.lines[2], 31, 42, tightpixel15.Font)

	if textEntry { //Cursor
		xStartPos := tightpixel15.Font.MeasureString(ic.lines[ic.line][:ic.index])
		_, widthChar := tightpixel15.Font.MeasureRune(rune(ic.lines[ic.line][ic.index]))
		if ic.line == 0 { //first line
			Line(img, margin+xStartPos, 25, margin+widthChar+xStartPos, 25)
		} else if ic.line == 1 { //second line
			Line(img, margin+xStartPos, 38, margin+widthChar+xStartPos, 38)
		} else { //third line
			Line(img, margin+xStartPos, 51, margin+widthChar+xStartPos, 51)
		}
	}
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

// Left moves cursor left
func (ic *InputChars) Left() byte {
	ic.index--
	if ic.index < 0 {
		ic.line--
		if ic.line < 0 {
			ic.line = len(ic.lines) - 1
		}
		ic.index = len(ic.lines[ic.line]) - 1
	}

	return ic.GetCurrent()
}

//Up moves cursor up a line
func (ic *InputChars) Up() byte {
	ic.line--
	if ic.line < 0 {
		ic.line = len(ic.lines) - 1
	}

	if ic.index >= len(ic.lines[ic.line]) {
		ic.index = len(ic.lines[ic.line]) - 1
	}

	return ic.GetCurrent()
}

//Down moves cursor down a line
func (ic *InputChars) Down() byte {
	ic.line++
	if ic.line >= len(ic.lines) {
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
	fmt.Println("c: ", c, "inputchar: ", ic.lines[ic.line][ic.index])
	for line := 0; line <= len(ic.lines)-1; line++ {
		for index := 0; index <= len(ic.lines[line])-1; index++ {
			if ic.lines[line][index] == c {
				ic.line, ic.index = line, index
			} else {
				ic.line, ic.index = len(ic.lines)-1, len(ic.lines[ic.line])-1
			}
		}
	}

}
