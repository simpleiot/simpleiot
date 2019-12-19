package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// DiagIPAddressScreen is used to display the IP address
type DiagIPAddressScreen struct {
	softKeys *SoftKeys
	state    *isdata.State
	config   *isdata.Config
	menu     *Menu
	edit     bool
}

// NewDiagIPAddressScreen and returns a HomeScreen
func NewDiagIPAddressScreen(state *isdata.State, config *isdata.Config) *DiagIPAddressScreen {
	return &DiagIPAddressScreen{
		softKeys: NewSoftKeys("back"),
		state:    state,
		config:   config,
		menu:     &Menu{},
	}
}

// Render updates the home screen, and provides an image
func (s *DiagIPAddressScreen) Render(img draw.Image) {
	Clear(img)

	s.menu.ResetItems()

	s.menu.AddItemScreen(s.state.NetworkState.InterfaceStatus.IP, ScreenIDNoChange)

	Heading(img, "IP Address")
	s.menu.Render(img)
	s.softKeys.Render(img, 0, 54)
}

// Key processes key inputs to this screen
func (s *DiagIPAddressScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1Hold:
		s.menu.ResetArrowPos() // return arrow to top of screen
		return ScreenIDHome, nil, true
	case isdata.KeySK1Release:
		s.menu.ResetArrowPos() // return arrow to top of screen
		return ScreenIDPrev, nil, true
	case isdata.KeyUp, isdata.KeyUpHold, isdata.KeyDown, isdata.KeyDownHold, isdata.KeyRight, isdata.KeyRightHold, isdata.KeyLeft, isdata.KeyLeftHold:
		return s.menu.Key(key)
	}

	return ScreenIDNoChange, nil, true
}
