package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// PanelTypeScreen is used to select the panel type
type PanelTypeScreen struct {
	softKeys *SoftKeys
	state    *isdata.State
	config   *isdata.Config
	menu     *Menu
}

// NewPanelTypeScreen initializes and returns a panel type screen
func NewPanelTypeScreen(state *isdata.State, config *isdata.Config) *PanelTypeScreen {

	// Find which panel type to initialize the arrow at
	var selectedIndex int
	switch config.PanelType {
	case isdata.PanelTypeStandardPivot:
		selectedIndex = 1
	case isdata.PanelTypeStandardPump:
		selectedIndex = 0
	case isdata.PanelTypeLindsay:
		selectedIndex = 2
	}

	return &PanelTypeScreen{
		softKeys: NewSoftKeys("back"),
		state:    state,
		config:   config,
		menu:     NewMenu(true, selectedIndex),
	}
}

// Render updates the home screen, and provides an image
func (s *PanelTypeScreen) Render(img draw.Image) {
	Clear(img)

	s.menu.ResetItems()

	// add menu items
	s.menu.AddItemSelect("Standard Pump", isdata.PanelTypeStandardPump, s.config.PanelType == isdata.PanelTypeStandardPump)
	s.menu.AddItemSelect("Standard Pivot", isdata.PanelTypeStandardPivot, s.config.PanelType == isdata.PanelTypeStandardPivot)
	s.menu.AddItemSelect("Vision", isdata.PanelTypeLindsay, s.config.PanelType == isdata.PanelTypeLindsay)

	// render
	s.menu.Render(img)
	Heading(img, "Panel Type")
	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *PanelTypeScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
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
