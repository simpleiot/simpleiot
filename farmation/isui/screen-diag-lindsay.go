package isui

import (
	"fmt"
	"image/draw"
	"time"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// DiagLindsayScreen is a diagnostics sub screen
type DiagLindsayScreen struct {
	softKeys *SoftKeys
	state    *isdata.State
	config   *isdata.Config
	menu     Menu
}

// NewDiagLindsayScreen gives new screen to screen.go
func NewDiagLindsayScreen(state *isdata.State, config *isdata.Config) *DiagLindsayScreen {
	menu := Menu{}

	return &DiagLindsayScreen{
		softKeys: NewSoftKeys("back"),
		state:    state,
		config:   config,
		menu:     menu,
	}
}

// Render updates the home screen, and provides an image
func (s *DiagLindsayScreen) Render(img draw.Image) {
	Clear(img)
	s.menu.ResetItems()

	status := fmt.Sprintf("0x%x", s.state.LindsayRegs.Status)
	state := fmt.Sprintf("0x%x", uint16(s.state.LindsayRegs.State))
	durLastUpdate := time.Now().Sub(s.state.LindsayLastUpdate)
	durLastUpdateS := fmt.Sprintf("%.1f min", durLastUpdate.Minutes())
	if durLastUpdate > 60*time.Minute {
		durLastUpdateS = "no data"
	}

	// Gpio's
	s.menu.AddItemString("status", status)
	s.menu.AddItemString("state", s.state.LindsayRegs.State.String())
	s.menu.AddItemString("state", state)
	s.menu.AddItemOnOff("water", s.state.LindsayRegs.WaterOn(), nil)
	s.menu.AddItemOnOff("irr running", s.state.LindsayRegs.IrrigatorRunning(), nil)
	s.menu.AddItemOnOff("accessory 1", s.state.LindsayRegs.Accessory1On(), nil)
	s.menu.AddItemOnOff("accessory 2", s.state.LindsayRegs.Accessory2On(), nil)
	s.menu.AddItemString("last update", durLastUpdateS)

	Heading(img, "Vision Panel Information")
	s.menu.Render(img)
	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *DiagLindsayScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1Hold: // Back key held -> Home screen
		s.menu.ResetArrowPos() // return arrow to top of screen
		return ScreenIDHome, nil, true
	case isdata.KeySK1Release: // Back
		s.menu.ResetArrowPos() // return arrow to top of screen
		return ScreenIDPrev, nil, true
	case isdata.KeyUp, isdata.KeyDown, isdata.KeyRight, isdata.KeyLeft, isdata.KeyEnter:
		return s.menu.Key(key)
	}

	return ScreenIDNoChange, nil, true
}
