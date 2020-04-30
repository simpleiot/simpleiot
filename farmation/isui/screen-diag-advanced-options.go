package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// DiagAdvancedOptionsScreen contains engineering/reset options
type DiagAdvancedOptionsScreen struct {
	softKeys *SoftKeys
	state    *isdata.State
	config   *isdata.Config
	menu     *Menu
	edit     bool
}

// NewDiagAdvancedOptionsScreen initializes and returns a HomeScreen
func NewDiagAdvancedOptionsScreen(state *isdata.State, config *isdata.Config) *DiagAdvancedOptionsScreen {
	return &DiagAdvancedOptionsScreen{
		softKeys: NewSoftKeys("back"),
		state:    state,
		config:   config,
		menu:     &Menu{},
	}
}

// Render updates the home screen, and provides an image
func (s *DiagAdvancedOptionsScreen) Render(img draw.Image) {
	Clear(img)

	s.menu.ResetItems()

	// Advanced Export Options
	s.menu.AddItemScreen("Storage Admin", ScreenIDStorage)
	s.menu.AddItemCommand("System Logs", "export", isdata.ExportSystemLogs{})
	s.menu.AddItemCommand("Config", "export", isdata.ExportConfig{})

	// Logging Enable
	s.menu.AddItemOnOff("Pulse logging", s.config.LogPulseData,
		isdata.UpdateLogPulseEnable(!s.config.LogPulseData))
	s.menu.AddItemOnOff("Flow logging", s.config.LogFlowData,
		isdata.UpdateLogFlowEnable(!s.config.LogFlowData))
	s.menu.AddItemOnOff("Pres logging", s.config.LogPressureData,
		isdata.UpdateLogPressureEnable(!s.config.LogPressureData))

	// Factory Reset
	s.menu.AddItemCommand("Factory Rst", "reset", isdata.FactoryReset{})

	Heading(img, "Advanced Options")
	s.menu.Render(img)
	s.softKeys.Render(img, 0, 54)
}

// Key processes key inputs to this screen
func (s *DiagAdvancedOptionsScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1Hold:
		s.menu.ResetArrowPos() // return arrow to top of screen
		return ScreenIDHome, nil, true
	case isdata.KeySK1Release:
		s.menu.ResetArrowPos() // return arrow to top of screen
		return ScreenIDPrev, nil, true
	case isdata.KeyUp, isdata.KeyUpHold, isdata.KeyDown, isdata.KeyDownHold, isdata.KeyRight, isdata.KeyRightHold, isdata.KeyLeft, isdata.KeyLeftHold, isdata.KeyEnter, isdata.KeyEnterHold:
		return s.menu.Key(key)
	}

	return ScreenIDNoChange, nil, true
}
