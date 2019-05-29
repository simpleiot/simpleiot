package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// DiagPulsesPresScreen is used to display and edit two integers: flow pulses per gallon and pressure setting
type DiagPulsesPresScreen struct {
	textEntryScreen *TextEntryScreen
	softKeys        *SoftKeys
	state           *isdata.State
	config          *isdata.Config
	menu            *Menu
	edit            bool
}

// NewDiagPulsesPresScreen initializes and returns a HomeScreen
func NewDiagPulsesPresScreen(state *isdata.State, config *isdata.Config) *DiagPulsesPresScreen {
	return &DiagPulsesPresScreen{
		softKeys:        NewSoftKeys("back", "edit"),
		state:           state,
		config:          config,
		menu:            NewMenu(),
		textEntryScreen: NewTextEntryScreen(false, true),
	}
}

// Render updates the home screen, and provides an image
func (s *DiagPulsesPresScreen) Render(img draw.Image) {
	Clear(img)

	s.menu.ResetItems()

	s.menu.AddItemScreen("Flow Pulses", ScreenIDNoChange)
	s.menu.AddItemScreen("Pres Set", ScreenIDNoChange)

	if s.edit { // render text entry screen
		s.textEntryScreen.Render(img)
	} else { // render regular screen
		Heading(img, "Flow Pulses and Pres Set")
		s.menu.Render(img)
		s.softKeys.Render(img, 0, 54)
	}
}

// Key processes key inputs to this screen
func (s *DiagPulsesPresScreen) Key(key isdata.Key) (ScreenID, int, bool) {
	if s.edit { // passes key inputs to textEntryScreen and follows returned commands
		command := s.textEntryScreen.Key(key)
		switch command {
		case TextEntryCommandNone: // do nothing
		case TextEntryCommandSave: //save
			s.exitEdit()
			return ScreenIDNoChange, isdata.UpdatePulsesPerGallon(s.textEntryScreen.GetTextEdit()), true

		case TextEntryCommandCancel: // cancel
			s.exitEdit()
		}
	} else {
		switch key {
		case isdata.KeySK1: // Back
			s.menu.ResetArrowPos() // return arrow to top of screen
			return ScreenIDMainMenu, nil, true
		case isdata.KeySK2: // Edit
			s.edit = true
			s.textEntryScreen.txtEdit = s.menu.Description()
			s.textEntryScreen.inputChars.IndexTo(s.textEntryScreen.txtEdit[s.textEntryScreen.cursorPos]) // move inputChars cursor to current pos in txtEdit
		case isdata.KeyUp, isdata.KeyDown, isdata.KeyRight, isdata.KeyLeft, isdata.KeyEnter:
			return s.menu.Key(key)
		}
	}

	return ScreenIDNoChange, nil, true
}

func (s *DiagPulsesPresScreen) exitEdit() {
	s.edit = false
	s.textEntryScreen.ExitEdit()
}
