package isui

import (
	"image/draw"
	"strconv"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// PulseOutputTestScreen holds fields for the screen that
// allows testing of the pulse output functionality
type PulseOutputTestScreen struct {
	softKeys        *SoftKeys
	state           *isdata.State
	config          *isdata.Config
	menu            *Menu
	textEntryScreen *TextEntryScreen
	edit            bool
}

// NewPulseOutputTestScreen returns a pointer to the type
func NewPulseOutputTestScreen(state *isdata.State, config *isdata.Config) *PulseOutputTestScreen {
	return &PulseOutputTestScreen{
		softKeys:        NewSoftKeys("back", "edit"),
		state:           state,
		config:          config,
		menu:            NewMenu(),
		textEntryScreen: NewTextEntryScreen(false, true),
	}
}

// Render draws the screen
func (s *PulseOutputTestScreen) Render(img draw.Image) {
	Clear(img)

	if s.edit {
		s.textEntryScreen.Render(img)
	} else {
		s.menu.ResetItems()

		Heading(img, "Test Pulse Output")

		s.menu.AddItemOnOff("Test Output", s.config.PulseOutputTestOn, isdata.UpdatePulseOutputTestOn(!s.config.PulseOutputTestOn))
		s.menu.AddItemInt("Flow Rate", s.config.PulseOutputTestFlowRate)

		s.menu.Render(img)

		if s.menu.GetArrowPos() == 0 {
			s.softKeys.SetHidden(SK2, true)
		} else {
			s.softKeys.SetHidden(SK2, false)
		}
		s.softKeys.Render(img, 0, 54)
	}
}

// Key processes keypad input to this screen
func (s *PulseOutputTestScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	if s.edit { // passes key inputs to textEntryScreen and follows returned commands
		command := s.textEntryScreen.Key(key)
		switch command {
		case TextEntryCommandNone: // do nothing
		case TextEntryCommandSave: //save
			s.exitEdit()
			value, _ := strconv.Atoi(s.textEntryScreen.GetTextEdit()) // convert edited string to integer
			switch s.menu.GetArrowPos() {
			case 1:
				return ScreenIDNoChange, isdata.UpdatePulseOutputTestFlowRate(value), true
			}
		case TextEntryCommandCancel: // cancel
			s.exitEdit()
		}
	} else {
		switch key {
		case isdata.KeySK1Hold: // Back key held -> Home screen
			s.menu.ResetArrowPos() // return arrow to top of screen
			return ScreenIDHome, nil, true
		case isdata.KeySK1Release: // Back
			s.menu.ResetArrowPos() // return arrow to top of screen
			return ScreenIDPrev, nil, true
		case isdata.KeySK2: // Edit
			switch s.menu.GetArrowPos() {
			case 1:
				s.enterEdit()
			}

		case isdata.KeyEnter:
			switch s.menu.GetArrowPos() {
			case 1:
				s.enterEdit()
			default:
				return s.menu.Key(key)
			}
		case isdata.KeyUp, isdata.KeyUpHold, isdata.KeyDown, isdata.KeyDownHold, isdata.KeyRight, isdata.KeyRightHold, isdata.KeyLeft, isdata.KeyLeftHold:
			return s.menu.Key(key)
		}
	}

	return ScreenIDNoChange, nil, true
}

func (s *PulseOutputTestScreen) exitEdit() {
	s.edit = false
	s.textEntryScreen.ExitEdit()
}

func (s *PulseOutputTestScreen) enterEdit() {
	s.edit = true

	// set the text being edited and the header label
	switch s.menu.GetArrowPos() {
	case 1:
		s.textEntryScreen.txtEdit = strconv.Itoa(s.config.PulseOutputTestFlowRate) // convert integer value into string to edit w/ text entry screen
		s.textEntryScreen.headerLabel = "Flow Rate"
	}

	s.textEntryScreen.inputChars.IndexTo(s.textEntryScreen.txtEdit[s.textEntryScreen.cursorPos]) // move inputChars cursor to current pos in txtEdit
}
