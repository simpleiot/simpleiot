package isui

import (
	"fmt"
	"image/draw"
	"strconv"

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

	PulsesPerGallonStr, PressureSettingStr := strconv.Itoa(s.config.PulsesPerGallon), strconv.Itoa(s.config.PressureSetting) // turn values from config into strings to display
	fmt.Println(s.config.PulsesPerGallon, s.config.PressureSetting)
	s.menu.AddItemScreen(PulsesPerGallonStr, ScreenIDNoChange)
	s.menu.AddItemScreen(PressureSettingStr, ScreenIDNoChange)

	if s.edit { // render text entry screen
		s.textEntryScreen.Render(img)
	} else { // render regular screen
		Heading(img, "Flow Pulses and Pres Set")
		s.menu.Render(img)
		s.softKeys.Render(img, 0, 54)
	}
}

// Key processes key inputs to this screen
func (s *DiagPulsesPresScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	if s.edit { // passes key inputs to textEntryScreen and follows returned commands
		command := s.textEntryScreen.Key(key)
		switch command {
		case TextEntryCommandNone: // do nothing
		case TextEntryCommandSave: //save
			s.exitEdit()
			value, convError := strconv.Atoi(s.textEntryScreen.GetTextEdit()) // convert edited string to integer
			fmt.Println(s.textEntryScreen.GetTextEdit(), value, convError, "\nMenuPos:", s.menu.GetArrowPos())
			switch s.menu.GetArrowPos() {
			case 0:
				return ScreenIDNoChange, isdata.UpdatePulsesPerGallon(value), true

			case 1:
				return ScreenIDNoChange, isdata.UpdatePressureSetting(value), true
			}
		case TextEntryCommandCancel: // cancel
			s.exitEdit()
		}
	} else {
		switch key {
		case isdata.KeySK1: // Back
			s.menu.ResetArrowPos() // return arrow to top of screen
			return ScreenIDDiagConfig, nil, true
		case isdata.KeySK2: // Edit
			s.edit = true
			s.textEntryScreen.txtEdit = s.menu.Description()

			// set the header label
			switch s.menu.GetArrowPos() {
			case 0:
				s.textEntryScreen.headerLabel = "Pulses per gal"
			case 1:
				s.textEntryScreen.headerLabel = "Pressure setting"
			}

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
