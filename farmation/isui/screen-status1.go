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
		softKeys: NewSoftKeys("home", "tank"),
		icons:    NewIcons(true, false, true),
		state:    state,
		config:   config,
	}
}

// Render updates the home screen, and provides an image
func (s *StatusScreen1) Render(img draw.Image) {
	Clear(img)
	x := 8
	DrawTxt(img, strconv.Itoa(int(s.state.CurrentTankVolume)), x, 7, agencyfbbold40.Font)
	DrawTxt(img, "GALLONS", x, 38, tightpixel15.Font)

	// Tank level graphic
	x = 88
	y := 21
	h := 30
	w := 30

	font := tightpixel15.Font
	capacity := strconv.Itoa(s.config.TankCapacity)
	DrawTxt(img, capacity, x-font.MeasureString(capacity)-2, y-3, font)
	DrawTxt(img, strconv.Itoa(0), x-8, y+h-3, font)

	// Empty tank
	Polyline(img,
		x, y,
		x+3, y-3,
		x+w-3, y-3,
		x+w, y,
		x+w, y+h,
		x, y+h,
		x, y)
	Line(img,
		x, y-1,
		x+2, y-3)
	Line(img,
		x+w-2, y-3,
		x+w, y-1)
	Rect(img, x+5, y-5, 5, 2)

	// Water level in tank
	var lev float64

	if s.config.TankCapacity != 0 {
		lev = s.state.CurrentTankVolume / float64(s.config.TankCapacity) * float64(h)
	}

	RectFilled(img, x+1, y+h-int(lev), w-1, int(lev))

	// icons
	// page indicator
	s.icons.SetPage("page indicator", 1) // set page indicator icon to status1
	s.icons.Render(img)

	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *StatusScreen1) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeyLeft, isdata.KeySK1:
		return ScreenIDHome, nil, true
	case isdata.KeyRight:
		return ScreenIDStatus2, nil, true
	case isdata.KeySK2:
		return ScreenIDTankMenu1, nil, true
	}

	return ScreenIDNoChange, nil, true
}
