package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// FieldMenuScreen is used to display status info
type FieldMenuScreen struct {
	textEntryScreen *TextEntryScreen
	softKeys        *SoftKeys
	state           *isdata.State
	config          *isdata.Config
	menu            *Menu
	edit            bool
}

// NewFieldMenuScreen initializes and returns a HomeScreen
func NewFieldMenuScreen(state *isdata.State, config *isdata.Config) *FieldMenuScreen {
	return &FieldMenuScreen{
		softKeys:        NewSoftKeys("back", "edit", "import"),
		state:           state,
		config:          config,
		menu:            NewMenu(),
		textEntryScreen: NewTextEntryScreen(),
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
		s.textEntryScreen.Render(img)
	} else {
		Heading(img, "Field Menu")
		s.menu.Render(img)
		s.softKeys.Render(img, 0, 54)
	}
}

// Key handles some key inputs and passes the rest to textEntryScreen
func (s *FieldMenuScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	if s.edit {
		switch key {
		case isdata.KeySK1: //save
			s.exitEdit()
			return ScreenIDNoChange, isdata.UpdateFieldName{
				Index: s.menu.GetArrowPos(),
				Name:  s.textEntryScreen.GetTextEdit(),
			}, true

		case isdata.KeySK4: // cancel
			s.exitEdit()
		case isdata.KeySK2, isdata.KeySK3:
			s.textEntryScreen.Key(key)
			/*case isdata.KeyEnter:
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
				//s.inputChars.IndexTo(s.txtEdit[s.cursorPos])
				s.inputChars.Left()
				s.txtEntry = true // show cursor in abc... selection
				s.enterTxt()
			case isdata.KeyRight:
				//s.inputChars.IndexTo(s.txtEdit[s.cursorPos])
				fmt.Println(s.txtEdit[s.cursorPos])
				s.inputChars.Right()
				s.txtEntry = true
				s.enterTxt()
			case isdata.KeyUp:
				s.inputChars.Up()
				s.txtEntry = true
				s.enterTxt()
			case isdata.KeyDown:
				s.inputChars.Down()
				s.txtEntry = true
				s.enterTxt()*/
		}
	} else {
		switch key {
		case isdata.KeySK1:
			s.menu.ResetArrowPos() // return arrow to top of screen
			return ScreenIDMainMenu, nil, true
		case isdata.KeySK2:
			s.edit = true
			s.textEntryScreen.txtEdit = s.menu.Description()
		case isdata.KeyUp, isdata.KeyDown, isdata.KeyRight, isdata.KeyLeft, isdata.KeyEnter:
			return s.menu.Key(key)
		}
	}

	return ScreenIDNoChange, nil, true
}

/*// enterText replaces letter in field name at cursorPos with letter in abc... selection at cursor2Pos
func (s *FieldMenuScreen) enterTxt() {
	byteEdit := []byte(s.txtEdit) // turn into a slice of bytes to edit
	byteEdit[s.cursorPos] = s.inputChars.GetCurrent()
	s.txtEdit = string(byteEdit)
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
}*/

func (s *FieldMenuScreen) exitEdit() {
	s.edit = false
}
