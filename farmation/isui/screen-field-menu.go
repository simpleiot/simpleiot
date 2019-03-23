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
			s.txtEntry = false
			s.cursorPos = 0  //reset field name cursor to pos 0
			s.cursor2Pos = 0 //reset abc... cursor to pos 0
		case isdata.KeyEnter:
			if s.txtEntry {
				s.txtEntry = false
				s.cursorPos++ //move cursor in field name
				if s.cursorPos >= len(s.txtEdit) {
					s.cursorPos = 0
				}
				s.cursor2Pos = 0 //reset abc... cursor to pos 0
			} else {
				s.txtEntry = true
			}
		case isdata.KeyLeft:
			if s.txtEntry { //if in txt entry mode
				s.cursor2Pos-- //move cursor in abc... selection
				if s.cursor2Pos < 0 {
					s.cursor2Pos = len(s.abc) - 1
				}
			} else { //else
				s.cursorPos-- //move cursor in field name
				if s.cursorPos < 0 {
					s.cursorPos = len(s.txtEdit) - 1
				}
			}
		case isdata.KeyRight:
			if s.txtEntry { //if in txt entry mode
				s.cursor2Pos++ //move cursor in abc... selection
				if s.cursor2Pos >= len(s.abc) {
					s.cursor2Pos = 0
				}
			} else { //else
				s.cursorPos++ //move cursor in field name
				if s.cursorPos >= len(s.txtEdit) {
					s.cursorPos = 0
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
