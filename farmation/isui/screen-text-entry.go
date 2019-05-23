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
	cursorPos  int
}

// NewTextEntryScreen creates a new text entry widget
func NewTextEntryScreen(alpha, numbers bool) *TextEntryScreen {
	return &TextEntryScreen{
		softKeys:   NewSoftKeys("done", "bkspc", "ABC", "cancel"),
		inputChars: NewInputChars(alpha, numbers),
	}
}

//Render renders the text entry screen
func (s *TextEntryScreen) Render(img draw.Image) {

	// Header -- text being edited
	txtStartX := DrawTxtCentered(img, s.txtEdit, 64, 2, tightpixel15.Font) //assign margin, draw text
	width := 116
	Rect(img, 64-width/2-2, 0, width+2, 13)

	// cursor
	widthString := tightpixel15.Font.MeasureString(s.txtEdit[:s.cursorPos])
	_, widthChar := tightpixel15.Font.MeasureRune(rune(s.txtEdit[s.cursorPos]))
	Line(img, txtStartX+widthString+widthChar/2-2, 11, txtStartX+widthString+widthChar/2+2, 11)

	// input characters
	s.inputChars.Render(img)

	// soft keys
	s.softKeys.Render(img, 0, 54)
}

// TextEntryCommand defines commands that may be returned from Key function
type TextEntryCommand int

// define TextEntryCommands
const (
	TextEntryCommandNone TextEntryCommand = iota
	TextEntryCommandSave
	TextEntryCommandCancel
)

//Key handles some key inputs and passes the rest to inputChars
func (s *TextEntryScreen) Key(key isdata.Key) TextEntryCommand {
	switch key {
	case isdata.KeySK1: // save
		return TextEntryCommandSave
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
			s.enterTxt()
		} else {
			s.inputChars.Caps(true)
			s.caps = true
			s.enterTxt()
		}
	case isdata.KeySK4: // cancel
		return TextEntryCommandCancel
	case isdata.KeyEnter:
		if s.cursorPos >= len(s.txtEdit)-1 { // if at end of txt
			if s.txtEdit[s.cursorPos:] == "\x00" { // if last char is null
				s.txtEdit = s.txtEdit[:s.cursorPos] // delete null char
				s.right()                           // and loop to beginning of txt
			} else {
				s.txtEdit += "\x00" // add null for new char
				s.right()
			}
		} else {
			s.right()
		}
		s.inputChars.IndexTo(s.txtEdit[s.cursorPos])
	case isdata.KeyRight, isdata.KeyLeft, isdata.KeyUp, isdata.KeyDown:
		//s.inputChars.IndexTo(s.txtEdit[s.cursorPos])
		s.inputChars.Key(key)
		s.enterTxt()
	}

	return TextEntryCommandNone
}

// GetTextEdit returns the text that is being edited
func (s *TextEntryScreen) GetTextEdit() string {
	return s.txtEdit
}

//ExitEdit resets the input char cursor and the txtEdit cursor to 0
func (s *TextEntryScreen) ExitEdit() {
	s.cursorPos, s.inputChars.line, s.inputChars.index = 0, 0, 0
}

func (s *TextEntryScreen) right() {
	s.cursorPos++
	if s.cursorPos >= len(s.txtEdit) {
		s.cursorPos = 0
	}
}

func (s *TextEntryScreen) enterTxt() {
	byteEdit := []byte(s.txtEdit) // turn into a slice of bytes to edit
	byteEdit[s.cursorPos] = s.inputChars.GetCurrent()
	s.txtEdit = string(byteEdit)
}
