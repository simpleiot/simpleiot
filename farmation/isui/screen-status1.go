package isui

import (
	"image/draw"
	"strconv"

	"github.com/simpleiot/simpleiot/farmation/fonts/agencyfbbold40"
	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// StatusScreen1 is used to display status info
type StatusScreen1 struct {
	softKeys *SoftKeys
	icons    *Icons
	state    *isdata.State
	config   *isdata.Config
}

// NewStatusScreen1 initializes and returns a HomeScreen
func NewStatusScreen1(state *isdata.State, config *isdata.Config) *StatusScreen1 {
	softKeys := SoftKeys{}
	softKeys.SetLabel(0, "home")

	return &StatusScreen1{
		softKeys: NewSoftKeys("home"),
		icons:    NewIcons(true, false, true),
		state:    state,
		config:   config,
	}
}

// Render updates the home screen, and provides an image
func (s *StatusScreen1) Render(img draw.Image) {
	Clear(img)
	DrawTxt(img, strconv.Itoa(int(s.state.BatchApplied)), 11, 7, agencyfbbold40.Font)
	DrawTxt(img, "GALLONS", 11, 38, tightpixel15.Font)
	x := 38
	y := 51
	presWidth := 19
	DrawTxt(img, strconv.FormatFloat(s.state.PressureMin, 'f', 0, 64), x, y, tightpixel15.Font)
	DrawTxt(img, strconv.FormatFloat(s.state.PressureAvg, 'f', 0, 64), x+presWidth, y, tightpixel15.Font)
	DrawTxt(img, strconv.FormatFloat(s.state.PressureMax, 'f', 0, 64), x+2*presWidth, y, tightpixel15.Font)
	DrawTxt(img, "PSI", x+3*presWidth, y, tightpixel15.Font)

	s.softKeys.Render(img, 0, 54)

	// icons
	// page indicator
	s.icons.SetPage("page indicator", 1) // set page indicator icon to status1

	// outputs and arm
	s.icons.SetOnOff("arm", s.config.Arm)
	s.icons.SetOnOff("pump", s.state.GpioRelayInjectorEn)
	s.icons.SetOnOff("shutdown", s.state.GpioRelayShutdownEn)

	s.icons.Render(img)
}

// Key processes keypad input to this screen
func (s *StatusScreen1) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeyLeft, isdata.KeySK1:
		return ScreenIDHome, nil, true
	case isdata.KeyRight:
		return ScreenIDStatus2, nil, true
	}

	return ScreenIDNoChange, nil, true
}
