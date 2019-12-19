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
	menu     Menu
}

// NewDiagOutputsScreen gives new screen to screen.go
func NewDiagOutputsScreen(state *isdata.State, config *isdata.Config) *DiagOutputsScreen {
	isdata.InitState(state)
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
	inj, aux, shtdwn := s.config.ManualRelayInj, s.config.ManualRelayAux, s.config.ManualRelayShutdown
	injS, auxS, shtdwnS := s.state.GpioRelayInjectorEn,
		s.state.GpioRelayAuxEn, s.state.GpioRelayShutdownEn

	s.menu.AddItemAutoOffOn("Injector: "+BoolToString(injS), inj, isdata.UpdateManualRelayInj(inj.GetMsg()))
	s.menu.AddItemAutoOffOn("Aux: "+BoolToString(auxS), aux, isdata.UpdateManualRelayAux(aux.GetMsg()))
	s.menu.AddItemAutoOffOn("Shutdown: "+BoolToString(shtdwnS), shtdwn, isdata.UpdateManualRelayShutdown(shtdwn.GetMsg()))
	s.menu.AddItemScreen("Pulse Test", ScreenIDPulseOutputTest)

	Heading(img, "Diagnostics Outputs")
	s.menu.Render(img)
	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *DiagOutputsScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1Hold: // Back key held -> Home screen
		s.menu.ResetArrowPos() // return arrow to top of screen
		// set all relays to auto mode
		return ScreenIDHome, isdata.UpdateManualRelayAll(int(isdata.RelayControlStateAuto)), true
	case isdata.KeySK1Release: // Back
		s.menu.ResetArrowPos() // return arrow to top of screen
		// set all relays to auto mode
		return ScreenIDPrev, isdata.UpdateManualRelayAll(int(isdata.RelayControlStateAuto)), true
	case isdata.KeyUp, isdata.KeyUpHold, isdata.KeyDown, isdata.KeyDownHold, isdata.KeyRight, isdata.KeyRightHold, isdata.KeyLeft, isdata.KeyLeftHold, isdata.KeyEnter, isdata.KeyEnterHold:
		return s.menu.Key(key)
	}

	return ScreenIDNoChange, nil, true
}
