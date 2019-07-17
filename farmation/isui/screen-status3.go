package isui

import (
	"image/draw"
	"strconv"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// StatusScreen3 is used to display status info
type StatusScreen3 struct {
	softKeys *SoftKeys
	icons    *Icons
	state    *isdata.State
	config   *isdata.Config
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
	y1, y2, y3 := 8, 22, 36

	DrawTxt(img, "Min Pressure: ", x, y1, tightpixel15.Font)
	DrawTxt(img, "Avg Pressure: ", x, y2, tightpixel15.Font)
	DrawTxt(img, "Max Pressure: ", x, y3, tightpixel15.Font)

	x = 77

	DrawTxt(img, strconv.FormatFloat(s.state.PressureMin, 'f', 0, 64), x, y1, tightpixel15.Font)
	DrawTxt(img, strconv.FormatFloat(s.state.PressureAvg, 'f', 0, 64), x, y2, tightpixel15.Font)
	DrawTxt(img, strconv.FormatFloat(s.state.PressureMax, 'f', 0, 64), x, y3, tightpixel15.Font)
	DrawTxt(img, "PSI", x+20, y1, tightpixel15.Font)

	s.softKeys.Render(img, 0, 54)

	// icons
	s.icons.SetPage("page indicator", 3)
	s.icons.Render(img)
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
