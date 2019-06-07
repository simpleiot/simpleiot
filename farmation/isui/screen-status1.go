package isui

import (
	"fmt"
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
		icons:    NewIcons(true, true, true, true),
		state:    state,
		config:   config,
	}
}

// Render updates the home screen, and provides an image
func (s *StatusScreen1) Render(img draw.Image) {
	Clear(img)
	DrawTxt(img, strconv.Itoa(int(s.state.BatchApplied)), 11, 7, agencyfbbold40.Font)
	DrawTxt(img, "GALLONS", 11, 38, tightpixel15.Font)
	presTxt := fmt.Sprintf("%.1f %.1f %.1f PSI", s.state.PressureMin,
		s.state.PressureAvg, s.state.PressureMax)
	DrawTxt(img, presTxt, 38, 51, tightpixel15.Font)

	s.softKeys.Render(img, 0, 54)

	// icons
	s.icons.SetPage("page indicator", 1)
	s.icons.SetOnOff("arm", s.config.Arm)
	s.icons.SetOnOff("pump", s.state.GpioDigitalInjector)
	s.icons.SetOnOff("water", s.state.GpioDigitalWaterOn)
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
