package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/system"
)

// DiagSystemTimezoneScreen is used to configure system time
type DiagSystemTimezoneScreen struct {
	softKeys  *SoftKeys
	state     *isdata.State
	config    *isdata.Config
	menu      *Menu
	timezones []string
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

	ret := &DiagSystemTimezoneScreen{
		softKeys:  NewSoftKeys("back"),
		state:     state,
		config:    config,
		menu:      NewMenu(true, selectedIndex),
		timezones: timezones,
	}

	/*fmt.Println("COLLIN, timezones: ", ret.timezones)
	zoneDir, timezone, _ := system.GetTimezone()
	fmt.Println("COLLIN, current timezone: ", zoneDir, timezone)
	system.SetTimezone("US", "Eastern")
	zoneDir, timezone, _ = system.GetTimezone()
	fmt.Println("COLLIN, current timezone: ", zoneDir, timezone)
	system.SetTimezone("US", "Central")
	zoneDir, timezone, _ = system.GetTimezone()
	fmt.Println("COLLIN, current timezone: ", zoneDir, timezone)
	system.SetTimezone("America", "Nome")
	zoneDir, timezone, _ = system.GetTimezone()
	fmt.Println("COLLIN, current timezone: ", zoneDir, timezone)*/

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
		system.SetTimezone("US", s.config.Timezone)
		//system.SetTime(time.Now()) // Reset time with new timezone
		s.menu.ResetArrowPos()
		return ScreenIDHome, nil, true
	case isdata.KeySK1Release: // Back
		system.SetTimezone("US", s.config.Timezone)
		//system.SetTime(time.Now()) // Reset time with new timezone
		s.menu.ResetArrowPos()
		return ScreenIDPrev, nil, true
	case isdata.KeyEnter, isdata.KeyUp, isdata.KeyUpHold, isdata.KeyDown, isdata.KeyDownHold, isdata.KeyRight, isdata.KeyRightHold, isdata.KeyLeft, isdata.KeyLeftHold:
		return s.menu.Key(key)
	}
	return ScreenIDNoChange, nil, true
}
