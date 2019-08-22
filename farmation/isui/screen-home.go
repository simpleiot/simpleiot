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
	x := 31
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
			s.drawFlow(img, x, y, agencyfbbold40.Font)
		}
	} else {
		s.drawFlow(img, x, y, agencyfbbold40.Font)
	}

	// Flow window
	x = 80
	if s.config.OperatingMode != isdata.ISOperatingModeMonitor && s.config.Arm {
		DrawTxt(img, strconv.FormatFloat(highBound, 'f', 1, 64), x, 14, agencyfbbold20.Font)
		DrawTxt(img, strconv.FormatFloat(lowBound, 'f', 1, 64), x, 32, agencyfbbold20.Font)
	}

	s.softKeys.SetBlinking(SK4, s.state.FaultsActive.ActiveFaults())
	s.softKeys.Render(img, 0, 54)

	// icons
	// page indicator
	s.icons.SetPage("page indicator", 0) // set page indicator icon to home

	// outputs and arm
	s.icons.SetOnOff("arm", s.config.Arm)
	s.icons.SetOnOff("pump", s.state.GpioRelayInjectorEn)
	s.icons.SetOnOff("shutdown", s.state.GpioRelayShutdownEn)

	// inputs
	s.icons.SetOnOff("pump in", s.state.InputInjector == isdata.InputStateOn)
	s.icons.SetOnOff("water", s.state.InputWaterOn == isdata.InputStateOn)
	s.icons.SetOnOff("irrigator", s.state.InputIrrigator == isdata.InputStateOn)

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
func (s *HomeScreen) drawFlow(img draw.Image, x, y int, font *pixfont.PixFont) {
	rate := s.state.FlowRate
	if rate < 10 {
		DrawTxt(img, strconv.FormatFloat(rate, 'f', 1, 64), x, y, font)
	} else {
		DrawTxt(img, strconv.FormatFloat(rate, 'f', 0, 64), x, y, font)
	}
}
