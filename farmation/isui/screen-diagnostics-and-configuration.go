package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// DiagnosticsScreen diagnostics screen
type DiagnosticsScreen struct {
	softKeys *SoftKeys
	state    *isdata.State
	config   *isdata.Config
	arrowPos int
	menu     Menu
}

// NewDiagnosticsScreen gives new Diagnostics and Configuration screen to screen.go
func NewDiagnosticsScreen(state *isdata.State, config *isdata.Config) *DiagnosticsScreen {
	isdata.InitState(state) // !!! comment from sample screen !!! make sure that ProductStates and FieldStates arrays are large enough
	menu := Menu{}

	return &DiagnosticsScreen{
		// update this from sample screen
		softKeys: NewSoftKeys("back"),
		state:    state,
		config:   config,
		menu:     menu,
	}
}

// Render updates the home screen, and provides an image
func (s *DiagnosticsScreen) Render(img draw.Image) {
	Clear(img)
	s.menu.ResetItems()

	// Relays
	s.menu.AddItemOnOff("Inj Relay", s.config.ManualRelayInj,
		isdata.UpdateManualRelayInj(!s.config.ManualRelayInj))
	s.menu.AddItemOnOff("Aux Relay", s.config.ManualRelayAux,
		isdata.UpdateManualRelayAux(!s.config.ManualRelayAux))
	s.menu.AddItemOnOff("Shtdwn Relay", s.config.ManualRelayShutdown,
		isdata.UpdateManualRelayShutdown(!s.config.ManualRelayShutdown))

	// Gpio's
	s.menu.AddItemString("Injector", BoolToString(s.state.GpioDigitalInjector))
	s.menu.AddItemString("Irrigator", BoolToString(s.state.GpioDigitalIrrigator))
	s.menu.AddItemString("Water On", BoolToString(s.state.GpioDigitalWaterOn))
	s.menu.AddItemString("In", BoolToString(s.state.GpioDigitalIn))

	// Logging Enable
	s.menu.AddItemOnOff("Pulse logging", s.config.LogPulseData,
		isdata.UpdateLogPulseEnable(!s.config.LogPulseData))
	s.menu.AddItemOnOff("Flow logging", s.config.LogFlowData,
		isdata.UpdateLogFlowEnable(!s.config.LogFlowData))
	s.menu.AddItemOnOff("Press logging", s.config.LogPressureData,
		isdata.UpdateLogPressureEnable(!s.config.LogPressureData))

	s.menu.AddItemCommand("Reboot", isdata.Reboot{})
	Heading(img, "Diagnostics and Config")
	s.menu.Render(img)
	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *DiagnosticsScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1:
		s.menu.ResetArrowPos() // return arrow to top of screen
		return ScreenIDMainMenu, nil, true
	case isdata.KeyUp, isdata.KeyDown, isdata.KeyRight, isdata.KeyLeft, isdata.KeyEnter:
		return s.menu.Key(key)
	}

	return ScreenIDNoChange, nil, true
}
