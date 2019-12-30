package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/system"
)

// DiagSystemTimezoneScreen is used to configure system time
type DiagSystemTimezoneScreen struct {
	softKeys    *SoftKeys
	state       *isdata.State
	config      *isdata.Config
	menu        *Menu
	timezones   []string
	initialZone string
}

// NewDiagSystemTimezoneScreen initializes and returns a screen used to select a system timezone
func NewDiagSystemTimezoneScreen(state *isdata.State, config *isdata.Config) *DiagSystemTimezoneScreen {

	// Fetch list of possible US timezones
	timezones, _ := system.ReadTimezones("US")

	var selectedIndex int
	for i, timezone := range timezones {
		if timezone == config.Timezone {
			selectedIndex = i
		}
	}

	_, zone, _ := system.GetTimezone()

	ret := &DiagSystemTimezoneScreen{
		softKeys:    NewSoftKeys("back"),
		state:       state,
		config:      config,
		menu:        NewMenu(true, selectedIndex),
		timezones:   timezones,
		initialZone: zone,
	}

	return ret
}

// Render updates the home screen, and provides an image
func (s *DiagSystemTimezoneScreen) Render(img draw.Image) {

	Clear(img)

	s.menu.ResetItems()

	for _, timezone := range s.timezones {
		s.menu.AddItemSelect(timezone, isdata.UpdateTimezone(timezone), s.config.Timezone == timezone)
	}

	s.menu.Render(img)
	s.softKeys.Render(img, 0, 54)
}

// Key processes key inputs to this screen
func (s *DiagSystemTimezoneScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1Hold: // Home
		interfaceMsg := s.exit()
		return ScreenIDHome, interfaceMsg, true
	case isdata.KeySK1Release: // Back
		interfaceMsg := s.exit()
		return ScreenIDPrev, interfaceMsg, true
	case isdata.KeyEnter, isdata.KeyUp, isdata.KeyUpHold, isdata.KeyDown, isdata.KeyDownHold, isdata.KeyRight, isdata.KeyRightHold, isdata.KeyLeft, isdata.KeyLeftHold:
		return s.menu.Key(key)
	}
	return ScreenIDNoChange, nil, true
}

func (s *DiagSystemTimezoneScreen) exit() interface{} {
	s.menu.ResetArrowPos()
	system.SetTimezone("US", s.config.Timezone)
	if s.config.Timezone != s.initialZone {
		return isdata.RestartApp{}
	}

	return nil
}
