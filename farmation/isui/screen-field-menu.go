package isui

import (
	"fmt"
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// FieldMenuScreen is used to display status info
type FieldMenuScreen struct {
	softKeys     *SoftKeys
	softKeysEdit *SoftKeys
	state        *isdata.State
	config       *isdata.Config
	txtEdit      string
	arrowPos     int
	menu         Menu
	edit         bool
	caps         bool
	txtEntry     bool
	cursorPos    int
}

// NewFieldMenuScreen initializes and returns a HomeScreen
func NewFieldMenuScreen(state *isdata.State, config *isdata.Config) *FieldMenuScreen {
	menu := Menu{}
	menu.AddItemScreen("Field One", ScreenIDNoChange)
	menu.AddItemScreen("Field Two", ScreenIDNoChange)
	menu.AddItemScreen("Field Three", ScreenIDNoChange)
	menu.AddItemScreen("Field Four", ScreenIDNoChange)

	return &FieldMenuScreen{
		softKeys:     NewSoftKeys("back", "edit", "import"),
		softKeysEdit: NewSoftKeys("done", "bkspc", "ABC", "cancel"),
		state:        state,
		config:       config,
		menu:         menu,
	}
}

// Render updates the home screen, and provides an image
func (s *FieldMenuScreen) Render(img draw.Image) {
	Clear(img)
	if s.edit {
		fmt.Println(s.txtEdit)
		txtStartX := DrawTxtCentered(img, s.txtEdit, 64, 2, tightpixel15.Font)
		//fmt.Println("txtStartX: ", txtStartX)
		width := 116
		Rect(img, 64-width/2-2, 0, width+2, 13)

		s.softKeysEdit.Render(img, 0, 54)
		if s.caps {
			DrawTxt(img, "ABCDEFGHIJKLMNOPQRSTUVWX", 3, 16, tightpixel15.Font)
			DrawTxt(img, "YZ", 3, 29, tightpixel15.Font)
			DrawTxt(img, "123456789.", 3, 42, tightpixel15.Font)
		} else {
			DrawTxt(img, "abcdefghijklmnopqrstuvwxyz", 3, 16, tightpixel15.Font)
			DrawTxt(img, "123456789.", 3, 29, tightpixel15.Font)
		}
		widthString := int(tightpixel15.Font.MeasureString(s.txtEdit[:s.cursorPos]))
		widthChar := int(tightpixel15.Font.MeasureString(s.txtEdit[s.cursorPos : s.cursorPos+1]))
		fmt.Println(s.txtEdit[s.cursorPos : s.cursorPos+1])
		Line(img, txtStartX+widthString, 11, txtStartX+widthString+widthChar-1, 11)
		if s.txtEntry {

		}
	} else {
		Heading(img, "Field Menu")
		s.menu.Render(img)
		s.softKeys.Render(img, 0, 54)
	}
}

// Key processes keypad input to this screen
func (s *FieldMenuScreen) Key(key isdata.Key) (ScreenID, interface{}) {
	if s.edit {
		switch key {
		case isdata.KeySK2: //backspace
			if s.cursorPos >= len(s.txtEdit)-1 { //if cursor is at end of text
				if len(s.txtEdit) > 1 { //and if length of text is more than one character
					s.txtEdit = s.txtEdit[:s.cursorPos] //slice current char off end
				}
			} else { //if cursor is in middle or begginning
				s.txtEdit = s.txtEdit[:s.cursorPos] + s.txtEdit[s.cursorPos+1:] //splice two strings on either side of char
			}
			if s.cursorPos > 0 { //if text is more than one char
				s.cursorPos-- //move cursor back one space
				fmt.Println(s.cursorPos)
			}
		case isdata.KeySK3:
			if s.caps {
				s.caps = false
			} else {
				s.caps = true
			}
		case isdata.KeySK4:
			s.edit = false
			s.caps = false
		case isdata.KeyEnter:
			s.txtEntry = true
		case isdata.KeyLeft:
			if s.edit {
				s.cursorPos--
				if s.cursorPos < 0 {
					s.cursorPos = len(s.txtEdit) - 1
				}
				fmt.Println(s.cursorPos)
			}
		case isdata.KeyRight:
			if s.edit {
				s.cursorPos++
				if s.cursorPos >= len(s.txtEdit) {
					s.cursorPos = 0
				}
				fmt.Println(s.cursorPos)
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
		case isdata.KeyUp, isdata.KeyDown, isdata.KeyRight, isdata.KeyLeft, isdata.KeyEnter:
			return s.menu.Key(key)
		}
	}

	return ScreenIDNoChange, nil
}
