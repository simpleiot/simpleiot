package isui

import (
	"fmt"
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// DiagLindsayScreen is a diagnostics sub screen
type DiagLindsayScreen struct {
	softKeys *SoftKeys
	state    *isdata.State
	config   *isdata.Config
	arrowPos int
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

	status := fmt.Sprintf("0x%x", s.state.LindsayStatus)
	state := fmt.Sprintf("0x%x", s.state.LindsayState)

	// Gpio's
	s.menu.AddItemString("status", status)
	s.menu.AddItemString("state", state)

	Heading(img, "Lindsay Panel information")
	s.menu.Render(img)
	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *DiagLindsayScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1:
		s.menu.ResetArrowPos() // return arrow to top of screen
		return ScreenIDDiagConfig, nil, true
	case isdata.KeyUp, isdata.KeyDown, isdata.KeyRight, isdata.KeyLeft, isdata.KeyEnter:
		return s.menu.Key(key)
	}

	return ScreenIDNoChange, nil, true
}
