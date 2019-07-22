package isui

import (
	"image/draw"
	"strconv"

	"github.com/simpleiot/simpleiot/farmation/fonts/agencyfbbold40"
	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// StatusScreen1 is used to display status info
type StatusScreen1 struct {
	softKeys *SoftKeys
	icons    *Icons
	state    *isdata.State
	config   *isdata.Config
}

// NewStatusScreen1 initializes and returns a HomeScreen
func NewStatusScreen1(state *isdata.State, config *isdata.Config) *StatusScreen1 {
	softKeys := SoftKeys{}
	softKeys.SetLabel(0, "home")

	return &StatusScreen1{
		softKeys: NewSoftKeys("home"),
		icons:    NewIcons(true, false, true),
		state:    state,
		config:   config,
	}
}

// Render updates the home screen, and provides an image
func (s *StatusScreen1) Render(img draw.Image) {
	Clear(img)
	DrawTxt(img, strconv.Itoa(int(s.config.TankCapacity-int(s.state.CurrentTankVolume))), 4, 7, agencyfbbold40.Font)
	DrawTxt(img, "GALLONS", 4, 38, tightpixel15.Font)

	s.softKeys.Render(img, 0, 54)

	// Tank level graphic
	x := 78
	y := 21
	h := 30
	w := 30

	font := tightpixel15.Font
	capacity := strconv.Itoa(s.config.TankCapacity)
	DrawTxt(img, capacity, x-font.MeasureString(capacity)-2, y-3, font)
	DrawTxt(img, strconv.Itoa(0), x-8, y+h-3, font)

	Polyline(img,
		x, y,
		x+3, y-3,
		x+w-3, y-3,
		x+w, y,
		x+w, y+h,
		x, y+h,
		x, y)

	Rect(img, x+5, y-5, 5, 2)

	lev := s.state.CurrentTankVolume / float64(s.config.TankCapacity) * float64(h)

	RectFilled(img, x+1, y+h-int(lev), w-1, int(lev))

	// icons
	// page indicator
	s.icons.SetPage("page indicator", 1) // set page indicator icon to status1

	// outputs and arm
	s.icons.SetOnOff("arm", s.config.Arm)
	s.icons.SetOnOff("pump", s.state.GpioRelayInjectorEn)
	s.icons.SetOnOff("shutdown", s.state.GpioRelayShutdownEn)

	s.icons.Render(img)
}

// Key processes keypad input to this screen
func (s *StatusScreen1) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeyLeft, isdata.KeySK1:
		return ScreenIDHome, nil, true
	case isdata.KeyRight:
		return ScreenIDStatus2, nil, true
	}

	return ScreenIDNoChange, nil, true
}
