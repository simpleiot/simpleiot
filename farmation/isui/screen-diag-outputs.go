package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// DiagOutputsScreen is a diagnostics sub screen
type DiagOutputsScreen struct {
	softKeys *SoftKeys
	state    *isdata.State
	config   *isdata.Config
	arrowPos int
	menu     Menu
}

// NewDiagOutputsScreen gives new screen to screen.go
func NewDiagOutputsScreen(state *isdata.State, config *isdata.Config) *DiagOutputsScreen {
	isdata.InitState(state) // !!! comment from sample screen !!! make sure that ProductStates and FieldStates arrays are large enough
	menu := Menu{}

	return &DiagOutputsScreen{
		softKeys: NewSoftKeys("back"),
		state:    state,
		config:   config,
		menu:     menu,
	}
}

// Render updates the home screen, and provides an image
func (s *DiagOutputsScreen) Render(img draw.Image) {
	Clear(img)
	s.menu.ResetItems()

	// Relays
	s.menu.AddItemOnOff("Injector", s.config.ManualRelayInj, isdata.UpdateManualRelayInj(!s.config.ManualRelayInj))
	s.menu.AddItemOnOff("Aux", s.config.ManualRelayAux, isdata.UpdateManualRelayAux(!s.config.ManualRelayAux))
	s.menu.AddItemOnOff("Shutdown", s.config.ManualRelayShutdown, isdata.UpdateManualRelayShutdown(!s.config.ManualRelayShutdown))

	Heading(img, "Diagnostics Outputs")
	s.menu.Render(img)
	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *DiagOutputsScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1:
		s.menu.ResetArrowPos() // return arrow to top of screen
		return ScreenIDDiagConfig, nil, true

	case isdata.KeyUp, isdata.KeyDown, isdata.KeyRight, isdata.KeyLeft, isdata.KeyEnter:
		return s.menu.Key(key)
	}

	return ScreenIDNoChange, nil, true
}
