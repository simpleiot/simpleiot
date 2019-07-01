package isui

import (
	"image/draw"
	"strconv"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// DiagShutdownSettingsScreen is used to display and edit two integers: flow pulses per gallon and pressure setting
type DiagShutdownSettingsScreen struct {
	textEntryScreen *TextEntryScreen
	softKeys        *SoftKeys
	state           *isdata.State
	config          *isdata.Config
	menu            *Menu
	edit            bool
}

// NewDiagShutdownSettingsScreen initializes and returns a HomeScreen
func NewDiagShutdownSettingsScreen(state *isdata.State, config *isdata.Config) *DiagShutdownSettingsScreen {
	return &DiagShutdownSettingsScreen{
		softKeys:        NewSoftKeys("back", "edit"),
		state:           state,
		config:          config,
		menu:            NewMenu(),
		textEntryScreen: NewTextEntryScreen(false, true),
	}
}

// Render updates the home screen, and provides an image
func (s *DiagShutdownSettingsScreen) Render(img draw.Image) {
	Clear(img)

	s.menu.ResetItems()

	s.menu.AddItemFloat("Rec Irrig Off", s.config.IrrigatorOffMin)
	s.menu.AddItemFloat("Recognize Alarm", s.config.AlarmRecognizeSec)
	s.menu.AddItemFloat("FR Percent Low", s.config.LowWindowPerc)
	s.menu.AddItemFloat("Percent High", s.config.HighWindowPerc)
	s.menu.AddItemFloat("FlwRt GPH Low", s.config.ManualLowAlarmGPH)
	s.menu.AddItemFloat("FR GPH High", s.config.ManualHighAlarmGPH)

	if s.edit { // render text entry screen
		s.textEntryScreen.Render(img)
	} else { // render regular screen
		Heading(img, "Shutdown Settings")
		s.menu.Render(img)
		s.softKeys.Render(img, 0, 54)
	}
}

// Key processes key inputs to this screen
func (s *DiagShutdownSettingsScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	if s.edit { // passes key inputs to textEntryScreen and follows returned commands
		command := s.textEntryScreen.Key(key)
		switch command {
		case TextEntryCommandNone: // do nothing
		case TextEntryCommandSave: //save
			s.exitEdit()
			value, _ := strconv.Atoi(s.textEntryScreen.GetTextEdit()) // convert edited string to integer
			switch s.menu.GetArrowPos() {
			case 0:
				return ScreenIDNoChange, isdata.UpdateIrrigatorOffMin(float64(value)), true
			case 1:
				return ScreenIDNoChange, isdata.UpdateAlarmRecognizeSec(float64(value)), true
			case 2:
				return ScreenIDNoChange, isdata.UpdateLowWindowPerc(value), true
			case 3:
				return ScreenIDNoChange, isdata.UpdateHighWindowPerc(value), true
			case 4:
				return ScreenIDNoChange, isdata.UpdateManualLowAlarmGPH(value), true
			case 5:
				return ScreenIDNoChange, isdata.UpdateManualHighAlarmGPH(value), true
			}
		case TextEntryCommandCancel: // cancel
			s.exitEdit()
		}
	} else {
		switch key {
		case isdata.KeySK1: // Back
			s.menu.ResetArrowPos() // return arrow to top of screen
			return ScreenIDPrev, nil, true
		case isdata.KeySK2: // Edit
			s.edit = true

			// set the text being edited and the header label
			switch s.menu.GetArrowPos() {
			case 0:
				s.textEntryScreen.txtEdit = strconv.Itoa(int(s.config.IrrigatorOffMin)) // convert integer value into string to edit w/ text entry screen
				s.textEntryScreen.headerLabel = "Minutes"
			case 1:
				s.textEntryScreen.txtEdit = strconv.Itoa(int(s.config.AlarmRecognizeSec)) // convert integer value into string to edit w/ text entry screen
				s.textEntryScreen.headerLabel = "Seconds"
			case 2:
				s.textEntryScreen.txtEdit = strconv.Itoa(int(s.config.LowWindowPerc)) // convert integer value into string to edit w/ text entry screen
				s.textEntryScreen.headerLabel = "FR Low Percentage"
			case 3:
				s.textEntryScreen.txtEdit = strconv.Itoa(int(s.config.HighWindowPerc)) // convert integer value into string to edit w/ text entry screen
				s.textEntryScreen.headerLabel = "High Percentage"
			case 4:
				s.textEntryScreen.txtEdit = strconv.Itoa(int(s.config.ManualLowAlarmGPH)) // convert integer value into string to edit w/ text entry screen
				s.textEntryScreen.headerLabel = "Flow Rate Low GPH"
			case 5:
				s.textEntryScreen.txtEdit = strconv.Itoa(int(s.config.ManualHighAlarmGPH)) // convert integer value into string to edit w/ text entry screen
				s.textEntryScreen.headerLabel = "FR High GPH"
			}

			s.textEntryScreen.inputChars.IndexTo(s.textEntryScreen.txtEdit[s.textEntryScreen.cursorPos]) // move inputChars cursor to current pos in txtEdit
		case isdata.KeyUp, isdata.KeyDown, isdata.KeyRight, isdata.KeyLeft, isdata.KeyEnter:
			return s.menu.Key(key)
		}
	}

	return ScreenIDNoChange, nil, true
}

func (s *DiagShutdownSettingsScreen) exitEdit() {
	s.edit = false
	s.textEntryScreen.ExitEdit()
}
