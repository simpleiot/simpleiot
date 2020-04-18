package isui

import (
	"image/draw"
	"strconv"
	"time"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// StatusScreen3 is used to display status info
type StatusScreen3 struct {
	softKeys *SoftKeys
	icons    *Icons
	state    *isdata.State
	config   *isdata.Config

	// blinking min pressure
	presLastBlink time.Time
	presOn        bool
}

// NewStatusScreen3 initializes and returns a HomeScreen
func NewStatusScreen3(state *isdata.State, config *isdata.Config) *StatusScreen3 {
	return &StatusScreen3{
		softKeys: NewSoftKeys("home"),
		icons:    NewIcons(true, false, false),
		state:    state,
		config:   config,
	}
}

// Render updates the home screen, and provides an image
func (s *StatusScreen3) Render(img draw.Image) {
	Clear(img)

	x := 2
	y1, y2, y3 := 10, 24, 38

	DrawTxt(img, "Current Base Pres: ", x, y1, tightpixel15.Font)
	DrawTxt(img, "Current Max Pres: ", x, y2, tightpixel15.Font)
	DrawTxt(img, "Shtdwn Threshold: ", x, y3, tightpixel15.Font)

	x = 90

	// Pressure values
	min := strconv.FormatFloat(s.state.PressureMin, 'f', 0, 64)

	// Blinking min pressure
	if s.config.Arm &&
		s.config.PressureShutdownEnabled &&
		s.state.GpioRelayInjectorEn &&
		s.state.PressureMin < s.config.PressureShutdownLow { // if injector pump is on and min pressure is low
		if time.Since(s.presLastBlink) >= 490*time.Millisecond {
			s.presLastBlink = time.Now()
			s.presOn = !s.presOn
		}
		if s.presOn {
			DrawTxt(img, min, x, y1, tightpixel15.Font)
		}
	} else {
		DrawTxt(img, min, x, y1, tightpixel15.Font)
	}

	// Max and shutdown pressures
	DrawTxt(img, strconv.FormatFloat(s.state.PressureMax, 'f', 0, 64), x, y2, tightpixel15.Font)
	if s.config.Arm && s.config.PressureShutdownEnabled {
		DrawTxt(img, strconv.FormatFloat(s.config.PressureShutdownLow, 'f', 0, 64), x, y3, tightpixel15.Font)
	} else {
		DrawTxt(img, "--", x, y3, tightpixel15.Font)
	}

	xOffSet := 18

	DrawTxt(img, "PSI", x+xOffSet, y1, tightpixel15.Font)
	DrawTxt(img, "PSI", x+xOffSet, y2, tightpixel15.Font)
	DrawTxt(img, "PSI", x+xOffSet, y3, tightpixel15.Font)

	// icons
	// Page indicator
	s.icons.SetPage("page indicator", 3)
	s.icons.Render(img)

	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *StatusScreen3) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1:
		return ScreenIDHome, nil, true
	case isdata.KeyLeft:
		return ScreenIDStatus2, nil, true
	case isdata.KeyRight:
		return ScreenIDHome, nil, true
	}

	return ScreenIDNoChange, nil, true
}
