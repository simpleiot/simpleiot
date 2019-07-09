package isui

import (
	"image/draw"
	"strconv"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// OperatingModeSetupScreen is used to display and edit shutdown mode settings
type OperatingModeSetupScreen struct {
	textEntryScreen *TextEntryScreen
	softKeys        *SoftKeys
	state           *isdata.State
	config          *isdata.Config
	menu            *Menu
	edit            bool
}

// NewOperatingModeSetupScreen initializes and returns screen
func NewOperatingModeSetupScreen(state *isdata.State, config *isdata.Config) *OperatingModeSetupScreen {
	return &OperatingModeSetupScreen{
		softKeys:        NewSoftKeys("back", "edit"),
		state:           state,
		config:          config,
		menu:            NewMenu(),
		textEntryScreen: NewTextEntryScreen(false, true),
	}
}

// Render updates the home screen, and provides an image
func (s *OperatingModeSetupScreen) Render(img draw.Image) {
	Clear(img)

	s.menu.ResetItems()

	s.menu.AddItemFloat("Rec Flow Alm", s.config.AlarmRecognizeSec)
	s.menu.AddItemFloat("Rec Irrig Off", s.config.IrrigatorOffMin)
	s.menu.AddItemFloat("High Lev Alm", s.config.HighWindowPerc)
	s.menu.AddItemFloat("Low Lev Alm", s.config.LowWindowPerc)
	s.menu.AddItemFloat("Manual High", s.config.ManualHighAlarmGPH)
	s.menu.AddItemFloat("Manual Low", s.config.ManualLowAlarmGPH)
	s.menu.AddItemOnOff("Pres Shtdwn", s.config.PressureShutdownEnabled, isdata.UpdatePressureShutdownEnabled())
	s.menu.AddItemFloat("Pres Low", s.config.PressureShutdownLow)
	//s.menu.AddItemInt("Batch Amount", int(config.BatchAmount))
	//s.menu.AddItemInt("Batch Applied", 0)

	if s.edit { // render text entry screen
		s.textEntryScreen.Render(img)
	} else { // render regular screen
		Heading(img, "Operating Mode Setup")
		s.menu.Render(img)
		s.softKeys.Render(img, 0, 54)
	}
}

// Key processes key inputs to this screen
func (s *OperatingModeSetupScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	if s.edit { // passes key inputs to textEntryScreen and follows returned commands
		command := s.textEntryScreen.Key(key)
		switch command {
		case TextEntryCommandNone: // do nothing
		case TextEntryCommandSave: //save
			s.exitEdit()
			value, _ := strconv.Atoi(s.textEntryScreen.GetTextEdit()) // convert edited string to integer
			switch s.menu.GetArrowPos() {
			case 0:
				return ScreenIDNoChange, isdata.UpdateAlarmRecognizeSec(float64(value)), true
			case 1:
				return ScreenIDNoChange, isdata.UpdateIrrigatorOffMin(float64(value)), true
			case 2:
				return ScreenIDNoChange, isdata.UpdateHighWindowPerc(value), true
			case 3:
				return ScreenIDNoChange, isdata.UpdateLowWindowPerc(value), true
			case 4:
				return ScreenIDNoChange, isdata.UpdateManualHighAlarmGPH(value), true
			case 5:
				return ScreenIDNoChange, isdata.UpdateManualLowAlarmGPH(value), true
			case 6:
				return ScreenIDNoChange, isdata.UpdatePressureShutdownLow(value), true
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
			s.enterEdit()
		case isdata.KeyEnter: // Edit
			s.enterEdit()
		case isdata.KeyUp, isdata.KeyDown, isdata.KeyRight, isdata.KeyLeft:
			return s.menu.Key(key)
		}
	}

	return ScreenIDNoChange, nil, true
}

func (s *OperatingModeSetupScreen) exitEdit() {
	s.edit = false
	s.textEntryScreen.ExitEdit()
}

func (s *OperatingModeSetupScreen) enterEdit() {
	s.edit = true

	// set the text being edited and the header label
	switch s.menu.GetArrowPos() {
	case 0:
		s.textEntryScreen.txtEdit = strconv.Itoa(int(s.config.AlarmRecognizeSec)) // convert integer value into string to edit w/ text entry screen
		s.textEntryScreen.headerLabel = "Seconds"
	case 1:
		s.textEntryScreen.txtEdit = strconv.Itoa(int(s.config.IrrigatorOffMin)) // convert integer value into string to edit w/ text entry screen
		s.textEntryScreen.headerLabel = "Minutes"
	case 2:
		s.textEntryScreen.txtEdit = strconv.Itoa(int(s.config.HighWindowPerc)) // convert integer value into string to edit w/ text entry screen
		s.textEntryScreen.headerLabel = "High Percent"
	case 3:
		s.textEntryScreen.txtEdit = strconv.Itoa(int(s.config.LowWindowPerc)) // convert integer value into string to edit w/ text entry screen
		s.textEntryScreen.headerLabel = "Low Percent"
	case 4:
		s.textEntryScreen.txtEdit = strconv.Itoa(int(s.config.ManualHighAlarmGPH)) // convert integer value into string to edit w/ text entry screen
		s.textEntryScreen.headerLabel = "High GPH"
	case 5:
		s.textEntryScreen.txtEdit = strconv.Itoa(int(s.config.ManualLowAlarmGPH)) // convert integer value into string to edit w/ text entry screen
		s.textEntryScreen.headerLabel = "Low GPH"
	case 6:
		s.textEntryScreen.txtEdit = strconv.Itoa(int(s.config.PressureShutdownLow)) // convert integer value into string to edit w/ text entry screen
		s.textEntryScreen.headerLabel = "Pres Low Percent"
	}

	s.textEntryScreen.inputChars.IndexTo(s.textEntryScreen.txtEdit[s.textEntryScreen.cursorPos]) // move inputChars cursor to current pos in txtEdit
}
