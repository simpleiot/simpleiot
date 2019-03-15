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
	arrowPos     int
	menu         Menu
	edit         bool
	caps         bool
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
		// Line(img, 10, 10, 40, 40)
		txt := s.menu.Description()
		txtStartX := DrawTxtCentered(img, txt, 64, 2, tightpixel15.Font)
		fmt.Println("txtStartX: ", txtStartX)
		width := 116
		Rect(img, 64-width/2-2, 0, width+2, 13)

		s.softKeysEdit.Render(img, 0, 54)
		if s.caps {
			DrawTxt(img, "123456789.", 3, 42, tightpixel15.Font)
		} else {
			DrawTxt(img, "abcdefghijklmnopqrstuvwxyz", 3, 16, tightpixel15.Font)
			DrawTxt(img, "123456789.", 3, 29, tightpixel15.Font)
		}
		//widthString := int(font.MeasureString(txt[:s.cursorPos]))
		//Line(img, txtStartX+widthString, 11, txtStartX+widthString+3, 11)
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
		case isdata.KeySK4:
			s.edit = false
			s.caps = false
		case isdata.KeySK3:
			if s.caps {
				s.caps = false
			} else {
				s.caps = true
			}
		case isdata.KeyDown:
			if s.edit {
				s.cursorPos--
				if s.cursorPos < 0 {
					s.cursorPos = len(s.menu.Description()) - 1
				}
				fmt.Println(s.cursorPos)
			}
		case isdata.KeyUp:
			if s.edit {
				s.cursorPos++
				if s.cursorPos >= len(s.menu.Description()) {
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
		case isdata.KeyUp, isdata.KeyDown, isdata.KeyRight, isdata.KeyLeft, isdata.KeyEnter:
			return s.menu.Key(key)
		}
	}

	return ScreenIDNoChange, nil
}
