package isui

import (
	"image/draw"
	"strconv"
	"time"

	"github.com/pbnjay/pixfont"
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

	// blinking flow rate
	flowLastBlink time.Time
	flowOn        bool
}

// NewHomeScreen initializes and returns a HomeScreen
func NewHomeScreen(state *isdata.State, config *isdata.Config) *HomeScreen {
	return &HomeScreen{
		softKeys: NewSoftKeys("menu", "mode", "pump", "faults"),
		icons:    NewIcons(true, true, true),
		state:    state,
		config:   config,
	}
}

// Render updates the home screen, and provides an image
func (s *HomeScreen) Render(img draw.Image) {
	Clear(img)
	highBound, lowBound := s.config.CalculateFlowWindow()
	x := 60
	y := 15

	// Flow rate
	if s.config.Arm &&
		s.state.GpioRelayInjectorEn &&
		s.state.FlowStatus == isdata.FlowStatusOffTarget { // if injector pump is on and flow rate is off target
		if time.Since(s.flowLastBlink) >= 490*time.Millisecond {
			s.flowLastBlink = time.Now()
			s.flowOn = !s.flowOn
		}
		if s.flowOn {
			s.drawVariablePrecision(img, s.state.FlowRate, x, y, agencyfbbold40.Font)
		}
	} else {
		s.drawVariablePrecision(img, s.state.FlowRate, x, y, agencyfbbold40.Font)
	}

	// Flow window
	x = 97
	if s.config.OperatingMode != isdata.ISOperatingModeMonitor && s.config.Arm {
		s.drawVariablePrecision(img, highBound, x, 14, agencyfbbold20.Font)
		s.drawVariablePrecision(img, lowBound, x, 32, agencyfbbold20.Font)
	}

	s.softKeys.SetBlinking(SK4, s.state.FaultsActive.ActiveFaults())
	s.softKeys.Render(img, 0, 54)

	// icons
	// page indicator
	s.icons.SetPage("page indicator", 0) // set page indicator icon to home

	// outputs and arm
	s.icons.SetOnOff("arm", s.config.Arm)
	s.icons.SetOnOff("injector", s.state.GpioRelayInjectorEn)
	s.icons.SetOnOff("shutdown", s.state.GpioRelayShutdownEn)

	// inputs
	s.icons.SetOnOff("injector_in", s.state.InputInjector == isdata.InputStateOn)
	s.icons.SetOnOff("water", s.state.InputWaterOn == isdata.InputStateOn)
	s.icons.SetOnOff("irrigator", s.state.InputIrrigator == isdata.InputStateOn)

	s.icons.Render(img)

	// signal icon
	x = 30
	y = 1

	switch {
	case s.state.NetworkState.InterfaceStatus.Rsrp >= 0: // Get rid of invalid signal case
		//case s.state.NetworkState.InterfaceStatus.Rsrp > -84: //Excellent
	case s.state.NetworkState.InterfaceStatus.Rsrp > -103: // Good
		Line(img, x+10, y, x+10, y+6)
		fallthrough
	case s.state.NetworkState.InterfaceStatus.Rsrp > -111: // Fair
		Line(img, x+8, y+2, x+8, y+6)
		fallthrough
	case s.state.NetworkState.InterfaceStatus.Rsrp > -130: // Poor
		Line(img, x+6, y+4, x+6, y+6)
		fallthrough
	case true: // Signal is a valid negative number
		// Tower symbol
		Line(img, x, y, x+4, y)
		Line(img, x+1, y+1, x+3, y+1)
		Line(img, x+2, y+2, x+2, y+6)
		// First bar
		Line(img, x+4, y+5, x+4, y+6)
	}

	// flow moving average window indicator. On if short window used
	if s.config.FlowAvgWindowShortUsed {
		x = 88
		y = 1
		w := 8
		h := 4
		Rect(img, x, y, w, h)
		Line(img, x, y+2, x+w, y+2)
		Line(img, x+4, y, x+4, y+h)
	}
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
	case isdata.KeySK2: // operating mode
		return ScreenIDOpMode1, nil, true
	case isdata.KeySK3: // pump
		return ScreenIDPumpMode, nil, true
	case isdata.KeySK4: // faults
		return ScreenIDFaultsActive, nil, true
	}
	return ScreenIDNoChange, nil, true
}

// if rate is < 10, draw with 1 floating point, otherwise with none
func (s *HomeScreen) drawVariablePrecision(img draw.Image, value float64, x, y int, font *pixfont.PixFont) {

	// If the value is less than 10 but would round up to
	// 10, we set it to 10 so that the displayed flow rate
	// isn't 10.0
	if value < 10 && value >= 9.95 {
		value = 10
	}

	if value < 10 {
		DrawTxtRight(img, strconv.FormatFloat(value, 'f', 1, 64), x, y, font)
	} else {
		DrawTxtRight(img, strconv.FormatFloat(value, 'f', 0, 64), x, y, font)
	}
}
