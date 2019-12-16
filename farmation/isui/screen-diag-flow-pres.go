package isui

import (
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

	s.menu.AddItemInt("Flw Pulses/Gal", s.config.PulsesPerGallon)
	s.menu.AddItemInt("FlwAvg Win Short", s.config.FlowAvgWindow)
	s.menu.AddItemInt("FlwAvg Window", s.config.FlowAvgWindowLong)
	s.menu.AddItemInt("FlwAvg PercDif", s.config.FlowAvgPercDiff)
	s.menu.AddItemInt("Pres Setting", s.config.PressureSetting)

	if s.edit { // render text entry screen
		s.textEntryScreen.Render(img)
	} else { // render regular screen
		Heading(img, "Flow and Pressure Settings")
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
			value, _ := strconv.Atoi(s.textEntryScreen.GetTextEdit()) // convert edited string to integer
			switch s.menu.GetArrowPos() {
			case 0:
				return ScreenIDNoChange, isdata.UpdatePulsesPerGallon(value), true

			case 1:
				return ScreenIDNoChange, isdata.UpdateFlowAvgWindow(value), true

			case 2:
				return ScreenIDNoChange, isdata.UpdateFlowAvgWindowLong(value), true
			case 3:
				return ScreenIDNoChange, isdata.UpdateFlowAvgPercDiff(value), true
			case 4:
				return ScreenIDNoChange, isdata.UpdatePressureSetting(value), true

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
			s.enterEdit()
		case isdata.KeyEnter: // Edit
			s.enterEdit()
		case isdata.KeyUp, isdata.KeyUpHold, isdata.KeyDown, isdata.KeyDownHold, isdata.KeyRight, isdata.KeyRightHold, isdata.KeyLeft, isdata.KeyLeftHold:
			return s.menu.Key(key)
		}
	}

	return ScreenIDNoChange, nil, true
}

func (s *DiagPulsesPresScreen) exitEdit() {
	s.edit = false
	s.textEntryScreen.ExitEdit()
}

func (s *DiagPulsesPresScreen) enterEdit() {
	s.edit = true
	// set the text being edited and the header label
	switch s.menu.GetArrowPos() {
	case 0:
		s.textEntryScreen.txtEdit = strconv.Itoa(s.config.PulsesPerGallon) // convert integer value into string to edit w/ text entry screen
		s.textEntryScreen.headerLabel = "Pulses/gal"
	case 1:
		s.textEntryScreen.txtEdit = strconv.Itoa(s.config.FlowAvgWindow) // convert integer value into string to edit w/ text entry screen
		s.textEntryScreen.headerLabel = "FlowAvg Win Short"
	case 2:
		s.textEntryScreen.txtEdit = strconv.Itoa(s.config.FlowAvgWindowLong) // convert integer value into string to edit w/ text entry screen
		s.textEntryScreen.headerLabel = "Flow Avg Window"
	case 3:
		s.textEntryScreen.txtEdit = strconv.Itoa(s.config.FlowAvgPercDiff) // convert integer value into string to edit w/ text entry screen
		s.textEntryScreen.headerLabel = "Flow Avg Percent Diff"
	case 4:
		s.textEntryScreen.txtEdit = strconv.Itoa(s.config.PressureSetting) // convert integer value into string to edit w/ text entry screen
		s.textEntryScreen.headerLabel = "Pressure setting"
	}
	// move inputChars cursor to current pos in txtEdit
	s.textEntryScreen.inputChars.IndexTo(s.textEntryScreen.txtEdit[s.textEntryScreen.cursorPos])
}
