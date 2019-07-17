package isui

import (
	"fmt"
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// DiagPanelScreen is a diagnostics sub screen
type DiagPanelScreen struct {
	softKeys *SoftKeys
	state    *isdata.State
	config   *isdata.Config
	menu     Menu
}

// NewDiagPanelScreen gives new screen to screen.go
func NewDiagPanelScreen(state *isdata.State, config *isdata.Config) *DiagPanelScreen {
	menu := Menu{}

	return &DiagPanelScreen{
		softKeys: NewSoftKeys("back"),
		state:    state,
		config:   config,
		menu:     menu,
	}
}

// Render updates the home screen, and provides an image
func (s *DiagPanelScreen) Render(img draw.Image) {
	Clear(img)
	s.menu.ResetItems()

	voltage := fmt.Sprintf("%.2fV", s.state.PanelDefinition.Voltage)

	s.menu.AddItemString("Type", s.state.PanelDefinition.Type.String())
	s.menu.AddItemString("Voltage", voltage)

	Heading(img, "Panel Detection")
	s.menu.Render(img)
	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *DiagPanelScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1: // Back
		s.menu.ResetArrowPos() // return arrow to top of screen
		return ScreenIDPrev, nil, true
	case isdata.KeyUp, isdata.KeyDown, isdata.KeyRight, isdata.KeyLeft, isdata.KeyEnter:
		return s.menu.Key(key)
	}

	return ScreenIDNoChange, nil, true
}
