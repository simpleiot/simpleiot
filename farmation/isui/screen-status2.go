package isui

import (
	"image/draw"
	"strconv"

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
		softKeys: NewSoftKeys("home", "mode", "pump"),
		icons:    NewIcons(true, false, true),
		state:    state,
		config:   config,
	}
}

// Render updates the home screen, and provides an image
func (s *StatusScreen2) Render(img draw.Image) {
	Clear(img)
	DrawTxt(img, "Field: "+s.config.FieldConfigs[s.config.CurrentFieldIndex].Description, 2, 8, tightpixel15.Font)
	DrawTxt(img, "Product: "+s.config.ProductConfigs[s.config.CurrentProductIndex].Description, 2, 22, tightpixel15.Font)
	DrawTxt(img, strconv.FormatFloat(s.state.FieldStates[s.config.CurrentFieldIndex][s.config.CurrentProductIndex].Total, 'f', 1, 64), 43, 37, tightpixel15.Font)

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
