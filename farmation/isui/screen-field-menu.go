package isui

import (
	"fmt"
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

type inputChars struct {
	Lines []string
	line  int
	index int
	caps  bool
}

func (ic *inputChars) Render(img draw.Image) {
}

func (ic *inputChars) Right() byte {
	ic.index++
	if ic.index >= len(ic.Lines[ic.line]) {
		ic.index = 0
		ic.line++
	}

	if ic.line >= len(ic.Lines) {
		ic.line = 0
	}

	return ic.GetCurrent()
}

func (ic *inputChars) GetCurrent() byte {
	return ic.Lines[ic.line][ic.index]
}

func (ic *inputChars) IndexTo(c byte) {
	for i := 0; i <= len(ic.Lines)-1; i++ {
		for j := 0; j <= len(ic.Lines[i]); j++ {
			if ic.Lines[i][j] == c {
				ic.line, ic.index = i, j
			}
		}
	}

}

var lowerCaseInput = inputChars{
	Lines: []string{"abcdefghijklm",
		"nopqrstuvwxyz",
		"0123456789. ",
	},
}

var upperCaseInput = inputChars{
	Lines: []string{"ABCDEFGHIJKL",
		"MNOPQRSTUVWXYZ",
		"0123456789. ",
	},
}

// FieldMenuScreen is used to display status info
type FieldMenuScreen struct {
	softKeys     *SoftKeys
	softKeysEdit *SoftKeys
	state        *isdata.State
	config       *isdata.Config
	txtEdit      string
	abc          string
	abcCaps      string
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

	s.menu.ResetItems()
	for _, fieldConfig := range s.config.FieldConfigs {
		s.menu.AddItemScreen(fieldConfig.Description, ScreenIDNoChange)
	}

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
		if s.caps { // ABC... selection
			DrawTxt(img, s.abcCaps[:13], 31, 16, tightpixel15.Font)
			DrawTxt(img, s.abcCaps[13:26], 31, 29, tightpixel15.Font)
			DrawTxt(img, s.abcCaps[26:], 31, 42, tightpixel15.Font)
		} else { // abc... selection
			DrawTxt(img, s.abc[:14], 31, 16, tightpixel15.Font)
			DrawTxt(img, s.abc[14:26], 31, 27, tightpixel15.Font)
			DrawTxt(img, s.abc[26:], 31, 42, tightpixel15.Font)
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
func (s *FieldMenuScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	if s.edit {
		switch key {
		case isdata.KeySK1: //save
			s.exitEdit()
			return ScreenIDNoChange, isdata.UpdateFieldName{
				Index: s.menu.GetArrowPos(),
				Name:  s.txtEdit,
			}, true

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
		case isdata.KeySK3: // ABC abc
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
		case isdata.KeySK4: // cancel
			s.exitEdit()
		case isdata.KeyEnter:
			s.txtEntry = false
			if s.cursorPos >= len(s.txtEdit)-1 { // if at end of txt
				if s.txtEdit[s.cursorPos:] == "\x00" { // if last char is null
					s.txtEdit = s.txtEdit[:s.cursorPos] // delete space
					s.cursorRight(false)                // and loop to beginning of txt
				} else {
					s.txtEdit += "\x00" // add null for new char
					s.cursorRight(false)
				}
			} else {
				s.cursorRight(false)
			}
			//fmt.Println(s.txtEdit[s.cursorPos : s.cursorPos+1])
		case isdata.KeyLeft:
			s.cursor2StartPos(s.cursorPos)
			s.txtEntry = true // show cursor in abc... selection
			s.cursorLeft(true)
			s.enterTxt()
		case isdata.KeyRight:
			s.cursor2StartPos(s.cursorPos)
			s.txtEntry = true
			s.cursorRight(true)
			s.enterTxt()
		case isdata.KeyUp:
			if s.cursor2Pos-len(s.abc)/3 >= 0 {
				s.cursor2Pos = s.cursor2Pos - len(s.abc)/3 //switch between lines of matrix formated abc... selection
				fmt.Println(s.cursor2Pos)
			}
		case isdata.KeyDown:
			if s.cursor2Pos+len(s.abc)/3 <= 30 {
				s.cursor2Pos = s.cursor2Pos + len(s.abc)/3
				fmt.Println(s.cursor2Pos, len(s.abc))
			} else {
				s.cursor2Pos = len(s.abc) - 1
			}
		}
	} else {
		switch key {
		case isdata.KeySK1:
			s.menu.ResetArrowPos() // return arrow to top of screen
			return ScreenIDMainMenu, nil, true
		case isdata.KeySK2:
			s.edit = true
			s.txtEdit = s.menu.Description()
			s.abc = "abcdefghijklmnopqrstuvwxyz123456789. "
			s.abcCaps = "ABCDEFGHIJKLMNOPQRSTUVWXYZ123456789. "
		case isdata.KeyUp, isdata.KeyDown, isdata.KeyRight, isdata.KeyLeft, isdata.KeyEnter:
			return s.menu.Key(key)
		}
	}

	return ScreenIDNoChange, nil, true
}

// enterText replaces letter in field name at cursorPos with letter in abc... selection at cursor2Pos
func (s *FieldMenuScreen) enterTxt() {
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

func (s *FieldMenuScreen) exitEdit() {
	s.edit = false
	s.caps = false
	s.txtEntry = false
	s.cursorPos = 0  //reset field name cursor to pos 0
	s.cursor2Pos = 0 //reset abc... cursor to pos 0
}

// cursorRight increments a cursor position - cursorPos if isCursor2 is false, cursor2Pos if true
func (s *FieldMenuScreen) cursorRight(isCursor2 bool) {
	cursorPos := &s.cursorPos
	txt := &s.txtEdit
	if isCursor2 {
		cursorPos = &s.cursor2Pos
		txt = &s.abc
	}
	(*cursorPos)++
	if *cursorPos >= len(*txt) {
		*cursorPos = 0
	}
}

// cursorLeft
func (s *FieldMenuScreen) cursorLeft(isCursor2 bool) {
	cursorPos := &s.cursorPos
	txt := &s.txtEdit
	if isCursor2 {
		cursorPos = &s.cursor2Pos
		txt = &s.abc
	}

	(*cursorPos)--
	if *cursorPos < 0 {
		*cursorPos = len(*txt) - 1
	}
}

// cursorStartPos
func (s *FieldMenuScreen) cursor2StartPos(cursorPos int) {
	char := ""
	if cursorPos >= len(s.txtEdit)-1 {
		char = s.txtEdit[cursorPos:]
	} else {
		char = s.txtEdit[cursorPos : cursorPos+1]
	}
	fmt.Println(char)
	for i := 0; i <= len(s.abc)-1; i++ {
		if s.abc[i:] == char || s.abcCaps[i:] == char { // in case at end of abc... string
			s.cursor2Pos = i
			break
		} else if s.abc[i:i+1] == char || s.abcCaps[i:i+1] == char { // at beginning or middle of abc... string
			s.cursor2Pos = i
			break
		} else {
			s.cursor2Pos = len(s.abc) - 1
		}
	}
}
