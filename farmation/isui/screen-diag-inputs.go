package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// DiagInputsScreen is a diagnostics sub screen
type DiagInputsScreen struct {
	softKeys *SoftKeys
	state    *isdata.State
	config   *isdata.Config
	menu     Menu
}

// NewDiagInputsScreen gives new screen to screen.go
func NewDiagInputsScreen(state *isdata.State, config *isdata.Config) *DiagInputsScreen {
	isdata.InitState(state) // !!! comment from sample screen !!! make sure that ProductStates and FieldStates arrays are large enough
	menu := Menu{}

	return &DiagInputsScreen{
		softKeys: NewSoftKeys("back", "reset"),
		state:    state,
		config:   config,
		menu:     menu,
	}
}

// Render updates the home screen, and provides an image
func (s *DiagInputsScreen) Render(img draw.Image) {
	Clear(img)
	s.menu.ResetItems()

	// Gpio's
	s.menu.AddItemString("Injector", BoolToString(s.state.GpioDigitalInjector))
	s.menu.AddItemString("Irrigator", BoolToString(s.state.GpioDigitalIrrigator))
	s.menu.AddItemString("Water", BoolToString(s.state.GpioDigitalWaterOn))
	s.menu.AddItemString("Aux In", BoolToString(s.state.GpioDigitalIn))
	s.menu.AddItemFloat("Ref Voltage", s.state.PressureVRef)
	s.menu.AddItemFloat("Pres. Voltage", s.state.PressureVSense)
	s.menu.AddItemInt("Flow Pulse Cnt", s.state.FlowPulseCount)

	Heading(img, "Diagnostics Inputs")
	s.menu.Render(img)

	if s.menu.GetArrowPos() != 6 { //Flow pulse count
		s.softKeys.SetHidden(SK2, true)
	} else {
		s.softKeys.SetHidden(SK2, false)
	}
	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *DiagInputsScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1Hold: // Back key held -> Home screen
		s.menu.ResetArrowPos() // return arrow to top of screen
		return ScreenIDHome, nil, true
	case isdata.KeySK1Release: // Back
		s.menu.ResetArrowPos() // return arrow to top of screen
		return ScreenIDPrev, nil, true
	case isdata.KeySK2: // reset
		switch s.menu.GetArrowPos() {
		case 6: // arrow is at flow pulse count menu item
			return ScreenIDNoChange, isdata.UpdateResetFlowPulseCount{}, true
		}
	case isdata.KeyUp, isdata.KeyUpHold, isdata.KeyDown, isdata.KeyDownHold, isdata.KeyRight, isdata.KeyRightHold, isdata.KeyLeft, isdata.KeyLeftHold, isdata.KeyEnter, isdata.KeyEnterHold:
		return s.menu.Key(key)
	}

	return ScreenIDNoChange, nil, true
}
