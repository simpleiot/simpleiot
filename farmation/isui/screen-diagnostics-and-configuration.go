package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/farmation/version"
)

// DiagnosticsScreen diagnostics screen
type DiagnosticsScreen struct {
	softKeys *SoftKeys
	state    *isdata.State
	config   *isdata.Config
	menu     Menu
}

// NewDiagnosticsScreen gives new Diagnostics and Configuration screen to screen.go
func NewDiagnosticsScreen(state *isdata.State, config *isdata.Config) *DiagnosticsScreen {
	isdata.InitState(state) // !!! comment from sample screen !!! make sure that ProductStates and FieldStates arrays are large enough
	menu := Menu{}

	return &DiagnosticsScreen{
		softKeys: NewSoftKeys("back"),
		state:    state,
		config:   config,
		menu:     menu,
	}
}

// Render updates the home screen, and provides an image
func (s *DiagnosticsScreen) Render(img draw.Image) {
	Clear(img)
	s.menu.ResetItems()

	// Sub screens
	s.menu.AddItemStringDown("OS Version", s.state.OSVersion.String())
	s.menu.AddItemStringDown("App Version", version.AppVersion)
	s.menu.AddItemStringDown("Serial Num", s.state.SerialNumber)
	s.menu.AddItemScreen("Device Name", ScreenIDDiagDevName)
	s.menu.AddItemScreen("System Time", ScreenIDDiagSystemTime)
	s.menu.AddItemScreen("Panel Type", ScreenIDPanelType)
	s.menu.AddItemScreen("Inputs", ScreenIDDiagInputs)
	s.menu.AddItemScreen("Outputs", ScreenIDDiagOutputs)
	s.menu.AddItemScreen("Flow and Pres", ScreenIDDiagPulsesPres)
	s.menu.AddItemScreen("Vision serial", ScreenIDDiagLindsay)
	s.menu.AddItemScreen("Network", ScreenIDModem)

	// Logging Enable
	s.menu.AddItemCommand("Data", "export", isdata.ExportData{})
	s.menu.AddItemOnOff("Pulse logging", s.config.LogPulseData,
		isdata.UpdateLogPulseEnable(!s.config.LogPulseData))
	s.menu.AddItemOnOff("Flow logging", s.config.LogFlowData,
		isdata.UpdateLogFlowEnable(!s.config.LogFlowData))
	s.menu.AddItemOnOff("Pres logging", s.config.LogPressureData,
		isdata.UpdateLogPressureEnable(!s.config.LogPressureData))

	s.menu.AddItemCommand("Reboot", "start", isdata.Reboot{})
	Heading(img, "Diagnostics and Config")
	s.menu.Render(img)
	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *DiagnosticsScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
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
