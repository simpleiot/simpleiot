package isui

import (
	"fmt"
	"image/draw"
	"strconv"

	"github.com/simpleiot/simpleiot/farmation/fonts/agencyfbbold20"
	"github.com/simpleiot/simpleiot/farmation/fonts/agencyfbbold40"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// HomeScreen is used to render the home screen
type HomeScreen struct {
	softKeys *SoftKeys
	icons    *Icons
	state    *isdata.State
	config   *isdata.Config
}

// NewHomeScreen initializes and returns a HomeScreen
func NewHomeScreen(state *isdata.State, config *isdata.Config) *HomeScreen {
	return &HomeScreen{
		softKeys: NewSoftKeys("menu", "mode", "pump"),
		icons:    NewIcons(true, true, true, true),
		state:    state,
		config:   config,
	}
}

// Render updates the home screen, and provides an image
func (s *HomeScreen) Render(img draw.Image) {
	Clear(img)
	rateS := strconv.FormatFloat(s.state.FlowRate, 'f', 1, 64)
	DrawTxt(img, rateS, 4, 12, agencyfbbold40.Font)
	DrawTxt(img, "963", 67, 11, agencyfbbold20.Font)
	DrawTxt(img, "963", 67, 29, agencyfbbold20.Font)

	s.softKeys.Render(img, 0, 54)

	// icons
	s.icons.SetOnOff("arm", s.config.Arm)
	s.icons.SetOnOff("pump", s.config.PumpAutoOff)
	s.icons.SetOnOff("water", s.state.GpioDigitalWaterOn)
	s.icons.SetPage("page indicator", 0) // set page indicator icon to home
	s.icons.Render(img)
}

// Key processes keypad input to this screen
func (s *HomeScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeyRight:
		return ScreenIDStatus1, nil, true
	case isdata.KeyLeft:
		return ScreenIDStatus3, nil, true
	case isdata.KeySK1: // menu
		return ScreenIDMainMenu, nil, true
	case isdata.KeySK3: // pump
		fmt.Println(!s.config.PumpAutoOff)
		return ScreenIDNoChange, isdata.UpdatePumpAutoOff(!s.config.PumpAutoOff), true
	}

	return ScreenIDNoChange, nil, true
}
