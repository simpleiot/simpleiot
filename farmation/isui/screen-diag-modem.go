package isui

import (
	"image/draw"
	"strconv"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// ModemScreen ethernet screen
type ModemScreen struct {
	softKeys *SoftKeys
	state    *isdata.State
	config   *isdata.Config
	menu     Menu
}

// NewModemScreen creates a modem information screen
func NewModemScreen(state *isdata.State, config *isdata.Config) *ModemScreen {
	isdata.InitState(state) // make sure that ProductStates and FieldStates arrays are large enough
	menu := Menu{}

	return &ModemScreen{
		// update this from sample screen
		softKeys: NewSoftKeys("back"),
		state:    state,
		config:   config,
		menu:     menu,
	}
}

// Render updates the screen, and provides an image
func (s *ModemScreen) Render(img draw.Image) {
	Clear(img)
	s.menu.ResetItems()

	Heading(img, "Modem")

	detected := "no"
	connected := "no"
	signal := strconv.Itoa(s.state.NetworkState.InterfaceStatus.Signal)
	rsrp := strconv.Itoa(s.state.NetworkState.InterfaceStatus.Rsrp)
	rsrq := strconv.Itoa(s.state.NetworkState.InterfaceStatus.Rsrq)
	errorCnt := strconv.Itoa(s.state.NetworkState.ErrorCnt)

	if s.state.NetworkState.InterfaceStatus.Detected {
		detected = "yes"
	}

	if s.state.NetworkState.InterfaceStatus.Connected {
		connected = "yes"
	}

	s.menu.AddItemString("Desc", s.state.NetworkState.Description)
	s.menu.AddItemString("Detected", detected)
	s.menu.AddItemString("Connected", connected)
	s.menu.AddItemStringRight("Signal", signal)
	s.menu.AddItemStringRight("RSRP", rsrp)
	s.menu.AddItemStringRight("RSRQ", rsrq)
	s.menu.AddItemStringRight("Network", s.state.NetworkState.InterfaceStatus.Operator)
	s.menu.AddItemScreen("IP", ScreenIDDiagIPAddress)
	s.menu.AddItemScreen("SIM/IMEI", ScreenIDDiagSIMImei)
	s.menu.AddItemStringRight("Err Count", errorCnt)
	s.menu.Render(img)
	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *ModemScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1Hold: // Back key held -> Home screen
		s.menu.ResetArrowPos() // return arrow to top of screen
		return ScreenIDHome, nil, true
	case isdata.KeySK1Release: // Back
		s.menu.ResetArrowPos() // return arrow to top of screen
		return ScreenIDPrev, nil, true
	case isdata.KeyUp, isdata.KeyUpHold, isdata.KeyDown, isdata.KeyDownHold, isdata.KeyRight, isdata.KeyRightHold, isdata.KeyLeft, isdata.KeyLeftHold, isdata.KeyEnter, isdata.KeyEnterHold:
		return s.menu.Key(key)
	}

	return ScreenIDNoChange, nil, true
}
