package isui

import (
	"image/draw"
	"strconv"
	"time"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// StatusScreen2 is used to display status info
type StatusScreen2 struct {
	softKeys *SoftKeys
	icons    *Icons
	state    *isdata.State
	config   *isdata.Config
}

// NewStatusScreen2 initializes and returns a HomeScreen
func NewStatusScreen2(state *isdata.State, config *isdata.Config) *StatusScreen2 {
	return &StatusScreen2{
		softKeys: NewSoftKeys("home"),
		icons:    NewIcons(true, false, true),
		state:    state,
		config:   config,
	}
}

// Render updates the home screen, and provides an image
func (s *StatusScreen2) Render(img draw.Image) {
	Clear(img)

	x := 2
	y1, y2, y3, y4 := 8, 19, 31, 42

	DrawTxt(img, s.config.FieldConfigs[s.config.CurrentFieldIndex].Description+" - "+s.config.ProductConfigs[s.config.CurrentProductIndex].Description, x, y1, tightpixel15.Font)
	DrawTxt(img, "Total: ", x, y2, tightpixel15.Font)
	DrawTxt(img, "Avg Flow: ", x, y3, tightpixel15.Font)
	DrawTxt(img, "over ", x, y4, tightpixel15.Font)

	x = 50

	DrawTxt(img, strconv.FormatFloat(s.state.FieldStates[s.config.CurrentFieldIndex][s.config.CurrentProductIndex].Total, 'f', 0, 64), x, y2, tightpixel15.Font)
	DrawTxt(img, "Gallons", x+21, y2, tightpixel15.Font)
	DrawTxt(img, strconv.FormatFloat(s.state.AvgFlowRate, 'f', 0, 64), x, y3, tightpixel15.Font)
	timeSinceArm := strconv.FormatFloat(time.Since(s.state.AvgFlowRateStart).Hours(), 'f', 1, 64)
	DrawTxtRight(img, timeSinceArm, x+31, y4, tightpixel15.Font)
	DrawTxt(img, "hrs", x+35, y4, tightpixel15.Font)

	s.softKeys.Render(img, 0, 54)

	// icons
	// page indicator
	s.icons.SetPage("page indicator", 2) // set page indicator icon to home

	// outputs and arm
	s.icons.SetOnOff("arm", s.config.Arm)
	s.icons.SetOnOff("pump", s.state.GpioRelayInjectorEn)
	s.icons.SetOnOff("shutdown", s.state.GpioRelayShutdownEn)

	s.icons.Render(img)
}

// Key processes keypad input to this screen
func (s *StatusScreen2) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1:
		return ScreenIDHome, nil, true
	case isdata.KeyLeft:
		return ScreenIDStatus1, nil, true
	case isdata.KeyRight:
		return ScreenIDStatus3, nil, true
	}

	return ScreenIDNoChange, nil, true
}
