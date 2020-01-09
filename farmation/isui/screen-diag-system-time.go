package isui

import (
	"image/draw"
	"time"

	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/system"
)

// DiagSystemTimeScreen is used to configure system time
type DiagSystemTimeScreen struct {
	timeEntryScreen *TimeEntryScreen
	softKeys        *SoftKeys
	state           *isdata.State
	config          *isdata.Config
	menu            *Menu
	edit            bool
}

// NewDiagSystemTimeScreen initializes and returns a HomeScreen
func NewDiagSystemTimeScreen(state *isdata.State, config *isdata.Config) *DiagSystemTimeScreen {
	ret := &DiagSystemTimeScreen{
		softKeys:        NewSoftKeys("back", "edit"),
		state:           state,
		config:          config,
		menu:            &Menu{},
		timeEntryScreen: NewTimeEntryScreen(),
	}

	return ret
}

// Render updates the home screen, and provides an image
func (s *DiagSystemTimeScreen) Render(img draw.Image) {
	Clear(img)

	s.menu.ResetItems()
	year, month, day := Date(time.Now(), false, false)
	hour, min, _ := Clock(time.Now(), false)
	s.menu.AddItemStringDown("Date", year+"/"+month+"/"+day)
	s.menu.AddItemStringDown("Time", hour+":"+min)
	s.menu.AddItemScreen("Timezone", ScreenIDDiagSystemTimezone)

	if s.edit { // render text entry screen
		s.timeEntryScreen.Render(img)
	} else { // render regular screen
		Heading(img, "System Time")
		s.menu.Render(img)

		if s.menu.GetArrowPos() == 2 {
			s.softKeys.SetHidden(SK2, true)
		} else {
			s.softKeys.SetHidden(SK2, false)
		}
		s.softKeys.Render(img, 0, 54)
	}
}

// Key processes key inputs to this screen
func (s *DiagSystemTimeScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	if s.edit { // passes key inputs to textEntryScreen and follows returned commands
		command := s.timeEntryScreen.Key(key)
		switch command {
		case TextEntryCommandNone: // do nothing
		case TextEntryCommandSave: //save
			s.exitEdit()
			system.SetTime(s.timeEntryScreen.GetTimeEdit())
			return ScreenIDNoChange, nil, true
		case TextEntryCommandCancel: // cancel
			s.exitEdit()
		}
	} else {
		switch key {
		case isdata.KeySK1Hold:
			s.menu.ResetArrowPos()
			return ScreenIDHome, nil, true
		case isdata.KeySK1Release:
			s.menu.ResetArrowPos()
			return ScreenIDPrev, nil, true
		case isdata.KeySK2: // Edit
			switch s.menu.GetArrowPos() {
			case 0:
				s.enterEdit(0)
			case 1:
				s.enterEdit(3)
			}
		case isdata.KeyEnter:
			switch s.menu.GetArrowPos() {
			case 0:
				s.enterEdit(0)
			case 1:
				s.enterEdit(3)
			default:
				return s.menu.Key(key)
			}
		case isdata.KeyUp, isdata.KeyUpHold, isdata.KeyDown, isdata.KeyDownHold, isdata.KeyRight, isdata.KeyRightHold, isdata.KeyLeft, isdata.KeyLeftHold:
			return s.menu.Key(key)
		}
	}

	return ScreenIDNoChange, nil, true
}

func (s *DiagSystemTimeScreen) exitEdit() {
	s.edit = false
	s.timeEntryScreen.ExitEdit()
}

func (s *DiagSystemTimeScreen) enterEdit(cursorPos int) {
	s.edit = true
	s.timeEntryScreen.InitTimeEdit()
	s.timeEntryScreen.InitCursorPos(cursorPos)
}
