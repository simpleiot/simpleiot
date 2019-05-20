package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

//TextEntryScreen ...
type TextEntryScreen struct {
	inputChars *InputChars
	softKeys   *SoftKeys
	txtEdit    string
	caps       bool
	txtEntry   bool
	cursorPos  int
}

// NewTextEntryScreen initializes and returns a HomeScreen
func NewTextEntryScreen() *TextEntryScreen {
	return &TextEntryScreen{
		softKeys:   NewSoftKeys("done", "bkspc", "ABC", "cancel"),
		inputChars: NewInputChars(true, true),
	}
}

//Render renders the text entry screen
func (s *TextEntryScreen) Render(img draw.Image) {

	// Header -- text being edited
	txtStartX := DrawTxtCentered(img, s.txtEdit, 64, 2, tightpixel15.Font)
	width := 116
	Rect(img, 64-width/2-2, 0, width+2, 13)

	if s.txtEntry { // cursor
		widthString := int(tightpixel15.Font.MeasureString(s.txtEdit[:s.cursorPos]))
		widthChar := int(tightpixel15.Font.MeasureString(s.txtEdit[s.cursorPos : s.cursorPos+1]))
		Line(img, txtStartX+widthString, 11, txtStartX+widthString+widthChar-1, 11)
	}

	// input characters
	s.inputChars.Render(img, s.txtEntry)

	// soft keys
	s.softKeys.Render(img, 0, 54)
}

//Key handles some key inputs to the screen and passes others to inputChars
func (s *TextEntryScreen) Key(key isdata.Key) {
	if key == isdata.KeySK3 || key == isdata.KeyRight || key == isdata.KeyLeft || key == isdata.KeyUp || key == isdata.KeyDown {
		s.inputChars.Key(key)
	} else {
		switch key {
		case isdata.KeySK2: // Backspace
			if s.cursorPos >= len(s.txtEdit)-1 { //if cursor is at end of text
				if len(s.txtEdit) > 1 { //and if length of text is more than one character
					s.txtEdit = s.txtEdit[:s.cursorPos] //cut current char off end
				}
			} else { //if cursor is in middle or begginning
				s.txtEdit = s.txtEdit[:s.cursorPos] + s.txtEdit[s.cursorPos+1:] //splice two strings on either side of char
			}
			if s.cursorPos > 0 { //if text is more than one char
				s.cursorPos-- //move cursor back one space
			}
		case isdata.KeySK3: // Caps
			if s.caps {
				s.inputChars.Caps(false)
				s.caps = false
				if s.txtEntry {
					//s.enterTxt()
				}
			} else {
				s.inputChars.Caps(true)
				s.caps = true
				if s.txtEntry {
					//s.enterTxt()
				}
			}
		}
	}
}

// GetTextEdit returns the text that is being edited
func (s *TextEntryScreen) GetTextEdit() string {
	return s.txtEdit
}
