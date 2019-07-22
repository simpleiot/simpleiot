package isui

import (
	"image/draw"
	"strconv"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// TankMenuScreen is used to display status info
type TankMenuScreen struct {
	textEntryScreen *TextEntryScreen
	softKeys        *SoftKeys
	state           *isdata.State
	config          *isdata.Config
	menu            Menu
	edit            bool
}

// NewTankMenuScreen initializes and returns a HomeScreen
func NewTankMenuScreen(state *isdata.State, config *isdata.Config) *TankMenuScreen {
	return &TankMenuScreen{
		softKeys:        NewSoftKeys("back", "full"),
		state:           state,
		config:          config,
		menu:            Menu{},
		textEntryScreen: NewTextEntryScreen(false, true),
	}
}

// Render updates the home screen, and provides an image
func (s *TankMenuScreen) Render(img draw.Image) {
	Clear(img)

	if s.edit {
		s.textEntryScreen.Render(img)
	} else {
		s.menu.ResetItems()
		s.menu.AddItemInt("Current Volume", int(s.state.CurrentTankVolume))
		s.menu.AddItemInt("Alert Level", int(s.config.TankAlertVolume))
		s.menu.AddItemInt("Tank Size", int(s.config.TankCapacity))
		s.menu.AddItemOnOff("Alert On/Off", s.config.TankAlertOn,
			isdata.UpdateTankAlertEnable(!s.config.TankAlertOn))

		s.menu.Render(img)

		Heading(img, "Tank Menu")
		s.softKeys.Render(img, 0, 54)
	}
}

// Key processes keypad input to this screen
func (s *TankMenuScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	if s.edit { // passes key inputs to textEntryScreen and follows returned commands
		command := s.textEntryScreen.Key(key)
		switch command {
		case TextEntryCommandNone: // do nothing
		case TextEntryCommandSave: //save
			s.exitEdit()
			value, _ := strconv.Atoi(s.textEntryScreen.GetTextEdit()) // convert edited string to integer
			switch s.menu.GetArrowPos() {
			case 1:
				return ScreenIDNoChange, isdata.UpdateTankAlertVolume(value), true
			case 2:
				return ScreenIDNoChange, isdata.UpdateTankCapacity(value), true
			}
		case TextEntryCommandCancel: // cancel
			s.exitEdit()
		}
	} else {
		switch key {
		case isdata.KeySK1: // Back
			s.menu.ResetArrowPos() // return arrow to top of screen
			return ScreenIDPrev, nil, true
		case isdata.KeySK2: // Full (tank is full)
			return ScreenIDNoChange, isdata.UpdateTankFull{}, true
		/*case isdata.KeySK3: // Edit
		switch s.menu.GetArrowPos() {
		case 0, 3: // A non-editable and an on/off selection -- do nothing
		default:
			s.enterEdit()
		}*/
		case isdata.KeyEnter: // Edit
			switch s.menu.GetArrowPos() {
			case 0:
			case 3:
				return s.menu.Key(key)
			default:
				s.enterEdit()
			}
		case isdata.KeyUp, isdata.KeyDown, isdata.KeyRight, isdata.KeyLeft:
			return s.menu.Key(key)
		}

	}
	return ScreenIDNoChange, nil, true
}

func (s *TankMenuScreen) exitEdit() {
	s.edit = false
	s.textEntryScreen.ExitEdit()
}

func (s *TankMenuScreen) enterEdit() {
	s.edit = true

	// set the text being edited and the header label
	switch s.menu.GetArrowPos() {
	case 1:
		s.textEntryScreen.txtEdit = strconv.Itoa(int(s.config.TankAlertVolume)) // convert integer value into string to edit w/ text entry screen
		s.textEntryScreen.headerLabel = "Alert Level"
	case 2:
		s.textEntryScreen.txtEdit = strconv.Itoa(int(s.config.TankCapacity)) // convert integer value into string to edit w/ text entry screen
		s.textEntryScreen.headerLabel = "Tank Size"
	}

	s.textEntryScreen.inputChars.IndexTo(s.textEntryScreen.txtEdit[s.textEntryScreen.cursorPos]) // move inputChars cursor to current pos in txtEdit
}
