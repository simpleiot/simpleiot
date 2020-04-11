package isui

import (
	"image/draw"
	"strconv"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// DiagPulsesPresScreen is used to display and edit two integers: flow pulses per gallon and pressure setting
type DiagPulsesPresScreen struct {
	textEntryScreen *TextEntryScreen
	helpContent     []isdata.HelpScreenContent
	softKeys        *SoftKeys
	state           *isdata.State
	config          *isdata.Config
	menu            *Menu
	edit            bool
}

// NewDiagPulsesPresScreen initializes and returns a HomeScreen
func NewDiagPulsesPresScreen(state *isdata.State, config *isdata.Config) *DiagPulsesPresScreen {
	return &DiagPulsesPresScreen{
		softKeys:        NewSoftKeys("back", "edit", "help"),
		state:           state,
		config:          config,
		menu:            &Menu{},
		textEntryScreen: NewTextEntryScreen(false, true),
	}
}

// Render updates the home screen, and provides an image
func (s *DiagPulsesPresScreen) Render(img draw.Image) {
	Clear(img)

	s.menu.ResetItems()

	helpPresSense := isdata.HelpScreenContent{
		Name: "Pressure Sense PSI",
		Text: "This is the pressure sensor maximum pressure. Default value is 300. Do not " +
			"modify unless sensor is replaced.",
	}

	s.menu.AddItemInt("Flo Pulses/Gal", s.config.PulsesPerGallon)
	//s.menu.AddItemStringRight("Avg Win Short", strconv.Itoa(s.config.FlowAvgWindow)+" s")
	s.menu.AddItemStringRight("Flo Avg Time", strconv.Itoa(s.config.FlowAvgWindowLong)+" s")
	//s.menu.AddItemStringRight("Flo Avg Diff", strconv.Itoa(s.config.FlowAvgPercDiff)+" %")
	s.menu.AddItemStringRight("Pres Sense", strconv.Itoa(s.config.PressureSetting)+" PSI",
		helpPresSense)
	s.menu.AddItemInt("Pulse Output K", s.config.PulseOutputK)
	s.menu.AddItemScreen("Pulse Test", ScreenIDPulseOutputTest)
	s.menu.AddItemStringRight("Sample Time", strconv.Itoa(s.config.SampleDuration)+" s")
	s.menu.AddItemStringRight("No Flo Timeout", strconv.Itoa(s.config.MaxNoPulseDuration)+" s")

	if s.edit { // render text entry screen
		s.textEntryScreen.Render(img)
	} else { // render regular screen
		Heading(img, "Flow and Pressure Settings")
		s.menu.Render(img)

		switch s.menu.GetArrowPos() {
		case 2:
			s.softKeys.SetHidden(2, false)
		default:
			s.softKeys.SetHidden(2, true)
		}
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

			//case 1:
			//	return ScreenIDNoChange, isdata.UpdateFlowAvgWindow(value), true

			case 1:
				return ScreenIDNoChange, isdata.UpdateFlowAvgWindowLong(value), true
			//case 3:
			//	return ScreenIDNoChange, isdata.UpdateFlowAvgPercDiff(value), true
			case 2:
				return ScreenIDNoChange, isdata.UpdatePressureSetting(value), true

			case 3:
				return ScreenIDNoChange, isdata.UpdatePulseOutputK(value), true
			case 4:
				// Pulse output test screen, do nothing
			case 5:
				return ScreenIDNoChange, isdata.UpdateSampleDuration(value), true
			case 6:
				return ScreenIDNoChange, isdata.UpdateMaxNoPulseDuration(value), true

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
		case isdata.KeySK3: // Help
			helpContent := s.menu.GetMenuItems()[s.menu.GetArrowPos()].Help
			if helpContent.Name == "" {
				break
			}
			return ScreenIDNoChange, helpContent, true
		case isdata.KeyEnter:
			switch s.menu.GetArrowPos() {
			case 4: // Open Pulse Output Test screen
				return s.menu.Key(key)
			default:
				s.enterEdit()
			}
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
		// convert integer value into string to edit w/ text entry screen

		s.textEntryScreen.txtEdit = strconv.Itoa(s.config.PulsesPerGallon)
		s.textEntryScreen.headerLabel = "Pulses/gal"
	//case 1:
	// convert integer value into string to edit w/ text entry screen
	//	s.textEntryScreen.txtEdit = strconv.Itoa(s.config.FlowAvgWindow)
	//	s.textEntryScreen.headerLabel = "Flo Avg Win Short"
	case 1:
		// convert integer value into string to edit w/ text entry screen
		s.textEntryScreen.txtEdit = strconv.Itoa(s.config.FlowAvgWindowLong)
		s.textEntryScreen.headerLabel = "Flow Avg Window"
	//case 3:
	// convert integer value into string to edit w/ text entry screen
	//	s.textEntryScreen.txtEdit = strconv.Itoa(s.config.FlowAvgPercDiff)
	//	s.textEntryScreen.headerLabel = "Flo Avg Percent Diff"
	case 2:
		// convert integer value into string to edit w/ text entry screen
		s.textEntryScreen.txtEdit = strconv.Itoa(s.config.PressureSetting)
		s.textEntryScreen.headerLabel = "Pressure Setting"
	case 3:
		s.textEntryScreen.txtEdit = strconv.Itoa(s.config.PulseOutputK)
		s.textEntryScreen.headerLabel = "Output Pulses/gal"
	case 4:
		// Pulse output test screen, do nothing
	case 5:
		s.textEntryScreen.txtEdit = strconv.Itoa(s.config.SampleDuration)
		s.textEntryScreen.headerLabel = "Sample Time"
	case 6:
		s.textEntryScreen.txtEdit = strconv.Itoa(s.config.MaxNoPulseDuration)
		s.textEntryScreen.headerLabel = "No Flow Timeout"
	}
	// move inputChars cursor to current pos in txtEdit
	s.textEntryScreen.inputChars.IndexTo(s.textEntryScreen.txtEdit[s.textEntryScreen.cursorPos])
}
