package isui

import (
	"image/draw"
	"time"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// DiagSystemTimeScreen is used to configure system time
type DiagSystemTimeScreen struct {
	textEntryScreen *TextEntryScreen
	softKeys        *SoftKeys
	state           *isdata.State
	config          *isdata.Config
	menu            *Menu
	edit            bool
}

// NewDiagSystemTimeScreen initializes and returns a HomeScreen
func NewDiagSystemTimeScreen(state *isdata.State, config *isdata.Config) *DiagSystemTimeScreen {
	return &DiagSystemTimeScreen{
		softKeys:        NewSoftKeys("back", "edit"),
		state:           state,
		config:          config,
		menu:            &Menu{},
		textEntryScreen: NewTextEntryScreen(true, true),
	}
}

// Render updates the home screen, and provides an image
func (s *DiagSystemTimeScreen) Render(img draw.Image) {
	Clear(img)

	s.menu.ResetItems()
	s.menu.AddItemTime("Time", time.Now())

	if s.edit { // render text entry screen
		s.textEntryScreen.Render(img)
	} else { // render regular screen
		Heading(img, "System Time")
		s.menu.Render(img)
		s.softKeys.Render(img, 0, 54)
	}
}

// Key processes key inputs to this screen
func (s *DiagSystemTimeScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	if s.edit { // passes key inputs to textEntryScreen and follows returned commands
		command := s.textEntryScreen.Key(key)
		switch command {
		case TextEntryCommandNone: // do nothing
		case TextEntryCommandSave: //save
			s.exitEdit()
			return ScreenIDNoChange, isdata.UpdateDevName(s.textEntryScreen.GetTextEdit()), true
		case TextEntryCommandCancel: // cancel
			s.exitEdit()
		}
	} else {
		switch key {
		case isdata.KeySK1Hold:
			s.menu.ResetArrowPos() // return arrow to top of screen
			return ScreenIDHome, nil, true
		case isdata.KeySK1Release:
			s.menu.ResetArrowPos() // return arrow to top of screen
			return ScreenIDPrev, nil, true
		case isdata.KeySK2: // Edit
			s.enterEdit()
		case isdata.KeyEnter: // Edit
			s.enterEdit()
		case isdata.KeyUp, isdata.KeyUpHold, isdata.KeyDown, isdata.KeyDownHold, isdata.KeyRight, isdata.KeyRightHold, isdata.KeyLeft, isdata.KeyLeftHold:
			return s.menu.Key(key)
		}
	}

	return ScreenIDNoChange, nil, true
}

func (s *DiagSystemTimeScreen) exitEdit() {
	s.edit = false
	s.textEntryScreen.ExitEdit()
}

func (s *DiagSystemTimeScreen) enterEdit() {
	s.edit = true
	// set the text being edited and the header label
	s.textEntryScreen.txtEdit = Clock(time.Now())
	// move inputChars cursor to current pos in txtEdit
	s.textEntryScreen.inputChars.IndexTo(s.textEntryScreen.txtEdit[s.textEntryScreen.cursorPos])
}
