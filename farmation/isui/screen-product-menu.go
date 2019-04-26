package isui

import (
	"fmt"
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// ProductMenuScreen
type ProductMenuScreen struct {
	softKeys     *SoftKeys
	softKeysEdit *SoftKeys
	state        *isdata.State
	config       *isdata.Config
	txtEdit      string
	abc          string
	abcCaps      string
	arrowPos     int
	menu         Menu
	edit         bool
	caps         bool
	txtEntry     bool
	cursorPos    int
	cursor2Pos   int
}

// NewProductMenuScreen initializes and returns a HomeScreen
func NewProductMenuScreen(state *isdata.State, config *isdata.Config) *ProductMenuScreen {
	menu := Menu{}
	menu.AddItemScreen("Product One", ScreenIDNoChange)
	menu.AddItemScreen("Product Two", ScreenIDNoChange)
	menu.AddItemScreen("Product Three", ScreenIDNoChange)
	menu.AddItemScreen("Product Four", ScreenIDNoChange)

	return &ProductMenuScreen{
		softKeys:     NewSoftKeys("back", "edit", "delete"),
		softKeysEdit: NewSoftKeys("done", "bkspc", "ABC", "cancel"),
		state:        state,
		config:       config,
		menu:         menu,
	}
}

// Render updates the home screen, and provides an image
func (s *ProductMenuScreen) Render(img draw.Image) {
	Clear(img)
	if s.edit {
		//fmt.Println(s.txtEdit)
		txtStartX := DrawTxtCentered(img, s.txtEdit, 64, 2, tightpixel15.Font)
		width := 116
		Rect(img, 64-width/2-2, 0, width+2, 13)

		s.softKeysEdit.Render(img, 0, 54)
		if s.txtEntry { //text entry mode
			if s.caps { //cursor for caps
				widthAbc := tightpixel15.Font.MeasureString(s.abcCaps[:s.cursor2Pos])
				widthAbcChar := tightpixel15.Font.MeasureString(s.abcCaps[s.cursor2Pos:s.cursor2Pos+1]) - 1
				widthLine1 := tightpixel15.Font.MeasureString(s.abcCaps[:24])
				widthLine2 := tightpixel15.Font.MeasureString(s.abcCaps[24:26])
				if s.cursor2Pos < 24 { //first line
					Line(img, 3+widthAbc, 25, 3+widthAbc+widthAbcChar, 25)
				} else if s.cursor2Pos < 26 { //second line
					Line(img, 3+widthAbc-widthLine1, 38, 3+widthAbc+widthAbcChar-widthLine1, 38)
				} else { //third line
					Line(img, 3+widthAbc-widthLine1-widthLine2, 51, 3+widthAbc+widthAbcChar-widthLine1-widthLine2, 51)
				}
			} else { //cursor for lowercase
				widthAbc := tightpixel15.Font.MeasureString(s.abc[:s.cursor2Pos])
				widthLine1 := tightpixel15.Font.MeasureString(s.abc[:26])
				widthAbcChar := tightpixel15.Font.MeasureString(s.abc[s.cursor2Pos:s.cursor2Pos+1]) - 1
				if s.cursor2Pos < 26 { //first line
					Line(img, 3+widthAbc, 25, 3+widthAbc+widthAbcChar, 25)
				} else { //second line
					Line(img, 3+widthAbc-widthLine1, 38, 3+widthAbc+widthAbcChar-widthLine1, 38)

				}
			}
		}
		if s.caps {
			DrawTxt(img, s.abcCaps[:24], 3, 16, tightpixel15.Font)
			DrawTxt(img, s.abcCaps[24:26], 3, 29, tightpixel15.Font)
			DrawTxt(img, s.abcCaps[26:], 3, 42, tightpixel15.Font)
		} else {
			DrawTxt(img, s.abc[:26], 3, 16, tightpixel15.Font)
			DrawTxt(img, s.abc[26:], 3, 29, tightpixel15.Font)
		}
		widthString := int(tightpixel15.Font.MeasureString(s.txtEdit[:s.cursorPos]))
		widthChar := int(tightpixel15.Font.MeasureString(s.txtEdit[s.cursorPos : s.cursorPos+1]))
		Line(img, txtStartX+widthString, 11, txtStartX+widthString+widthChar-1, 11)
	} else {
		Heading(img, "Product Menu")
		s.menu.Render(img)
		s.softKeys.Render(img, 0, 54)
	}
}

// Key processes keypad input to this screen
func (s *ProductMenuScreen) Key(key isdata.Key) (ScreenID, interface{}) {
	if s.edit {
		switch key {
		case isdata.KeySK2: //backspace
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
		case isdata.KeySK3:
			if s.caps {
				s.caps = false
				if s.txtEntry {
					s.enterTxt()
				}
			} else {
				s.caps = true
				if s.txtEntry {
					s.enterTxt()
				}
			}
		case isdata.KeySK4:
			s.edit = false
			s.caps = false
			s.txtEntry = false
			s.cursorPos = 0  //reset field name cursor to pos 0
			s.cursor2Pos = 0 //reset abc... cursor to pos 0
		case isdata.KeyEnter:
			if s.txtEntry {
				s.txtEntry = false
				if s.cursorPos == len(s.txtEdit)-1 {
					s.txtEdit = s.txtEdit + " "
					s.cursorRight(false)
				}
				fmt.Println(len(s.txtEdit)-1, s.cursorPos)
				s.cursorRight(false)
				fmt.Println(len(s.txtEdit)-1, s.cursorPos)
				s.cursor2Pos = 0 //reset abc... cursor to pos 0
				//s.caps = false //reset caps
			} else {
				s.txtEntry = true
				s.enterTxt()
			}
		case isdata.KeyLeft:
			if s.txtEntry { //if in txt entry mode
				s.cursorLeft(true)
				s.enterTxt()
			} else { //else
				fmt.Println(len(s.txtEdit)-1, s.cursorPos)
				s.cursorLeft(false)
				fmt.Println(len(s.txtEdit)-1, s.cursorPos)
			}
		case isdata.KeyRight:
			if s.txtEntry { //if in txt entry mode
				s.cursorRight(true)
				s.enterTxt()
			} else { //else
				fmt.Println(len(s.txtEdit)-1, s.cursorPos)
				s.cursorRight(false)
				fmt.Println(len(s.txtEdit)-1, s.cursorPos)
				if s.cursorPos == len(s.txtEdit)-1 {
					s.txtEdit = s.txtEdit + " "
					s.cursorRight(false)
				}
			}
		}
	} else {
		switch key {
		case isdata.KeySK1:
			s.menu.ResetArrowPos() // return arrow to top of screen
			return ScreenIDMainMenu, nil
		case isdata.KeySK2:
			s.edit = true
			s.txtEdit = s.menu.Description()
			s.abc = "abcdefghijklmnopqrstuvwxyz123456789."
			s.abcCaps = "ABCDEFGHIJKLMNOPQRSTUVWXYZ123456789."
		case isdata.KeyUp, isdata.KeyDown, isdata.KeyRight, isdata.KeyLeft, isdata.KeyEnter:
			return s.menu.Key(key)
		}
	}

	return ScreenIDNoChange, nil
}

// enterText replaces letter in field name at cursorPos with letter in abc... selection at cursor2Pos
func (s *ProductMenuScreen) enterTxt() {
	if s.cursorPos >= len(s.txtEdit)-1 { //if cursor is at end of text
		if s.caps {
			s.txtEdit = s.txtEdit[:s.cursorPos] + s.abcCaps[s.cursor2Pos:s.cursor2Pos+1]
		} else {
			s.txtEdit = s.txtEdit[:s.cursorPos] + s.abc[s.cursor2Pos:s.cursor2Pos+1]
		}
	} else { //if cursor is at beginning or middle of text
		if s.caps {
			s.txtEdit = s.txtEdit[:s.cursorPos] + s.abcCaps[s.cursor2Pos:s.cursor2Pos+1] + s.txtEdit[s.cursorPos+1:]
		} else {
			s.txtEdit = s.txtEdit[:s.cursorPos] + s.abc[s.cursor2Pos:s.cursor2Pos+1] + s.txtEdit[s.cursorPos+1:]
		}
	}
}

// cursorRight increments a cursor position - cursorPos if isCursor2 is false, cursor2Pos if true
func (s *ProductMenuScreen) cursorRight(isCursor2 bool) {
	cursorPos := &s.cursorPos
	txt := &s.txtEdit
	if isCursor2 {
		cursorPos = &s.cursor2Pos
		txt = &s.abc
	}
	(*cursorPos)++
	if *cursorPos >= len(*txt) {
		*cursorPos = len(*txt) - 1
	}
}

// cursorLeft
func (s *ProductMenuScreen) cursorLeft(isCursor2 bool) {
	cursorPos := &s.cursorPos
	if isCursor2 {
		cursorPos = &s.cursor2Pos
	}

	(*cursorPos)--
	if *cursorPos < 0 {
		*cursorPos = 0
	}
}
