package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// DiagSimImeiScreen is used to display modem information
type DiagSimImeiScreen struct {
	softKeys *SoftKeys
	state    *isdata.State
	config   *isdata.Config
	edit     bool
}

// NewDiagSimImeiScreen and returns a HomeScreen
func NewDiagSimImeiScreen(state *isdata.State, config *isdata.Config) *DiagSimImeiScreen {
	return &DiagSimImeiScreen{
		softKeys: NewSoftKeys("back"),
		state:    state,
		config:   config,
	}
}

// Render updates the home screen, and provides an image
func (s *DiagSimImeiScreen) Render(img draw.Image) {
	Clear(img)

	DrawTxtCentered(img, "IMEI: "+s.state.NetworkInterfaceConfig.Imei, 64, 20, tightpixel15.Font)
	DrawTxtCentered(img, "SIM: "+s.state.NetworkInterfaceConfig.Sim, 64, 40, tightpixel15.Font)

	Heading(img, "Modem Sim/IMEI")
	s.softKeys.Render(img, 0, 54)
}

// Key processes key inputs to this screen
func (s *DiagSimImeiScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1Hold:
		return ScreenIDHome, nil, true
	case isdata.KeySK1Release:
		return ScreenIDPrev, nil, true
	}

	return ScreenIDNoChange, nil, true
}
