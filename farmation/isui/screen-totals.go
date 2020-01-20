package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// TotalsScreen totals screen
type TotalsScreen struct {
	softKeys *SoftKeys
	state    *isdata.State
	config   *isdata.Config
	menu     Menu
}

// NewTotalsScreen gives new Totals screen to screen.go
func NewTotalsScreen(state *isdata.State, config *isdata.Config) *TotalsScreen {
	isdata.InitState(state)
	menu := Menu{}

	return &TotalsScreen{
		// update this from sample screen
		softKeys: NewSoftKeys("back", "reset"),
		state:    state,
		config:   config,
		menu:     menu,
	}
}

// Render updates the home screen, and provides an image
func (s *TotalsScreen) Render(img draw.Image) {
	Clear(img)

	Heading(img, s.config.FieldConfigs[s.config.CurrentFieldIndex].Description+" Totals")

	s.menu.ResetItems()
	s.menu.AddItemFloat(s.config.ProductConfigs[s.config.CurrentProductIndex].Description,
		s.state.FieldStates[s.config.CurrentFieldIndex][s.config.CurrentProductIndex].Total)
	s.menu.AddItemBreak("All Totals")
	s.menu.AddItemFloat("Total 1", s.state.Total1)
	s.menu.AddItemFloat("Total 2", s.state.Total2)
	s.menu.AddItemFloat("Lifetime Total", s.state.LifetimeTotal)
	s.menu.AddItemCommand("All Totals", "export", isdata.ExportFieldProductTotals{})

	s.menu.Render(img)

	if s.menu.GetArrowPos() == TotalScreenIndexLifetime || s.menu.GetArrowPos() == TotalScreenIndexExport {
		s.softKeys.SetHidden(SK2, true)
	} else {
		s.softKeys.SetHidden(SK2, false)
	}
	s.softKeys.Render(img, 0, 54)
}

// define position of various things on total screen
const (
	TotalScreenIndexCurrentProduct int = iota
	TotalScreenIndexMenuBreak
	TotalScreenIndexTotal1
	TotalScreenIndexTotal2
	TotalScreenIndexLifetime
	TotalScreenIndexExport
)

// Key processes keypad input to this screen
func (s *TotalsScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1Hold: // Back key held -> Home screen
		s.menu.ResetArrowPos() // return arrow to top of screen
		return ScreenIDHome, nil, true
	case isdata.KeySK1Release: // Back
		s.menu.ResetArrowPos() // return arrow to top of screen
		return ScreenIDPrev, nil, true
	case isdata.KeySK2: // Reset
		switch s.menu.GetArrowPos() {
		case TotalScreenIndexCurrentProduct:
			return ScreenIDNoChange, isdata.UpdateResetCurrentProduct{}, true
		case TotalScreenIndexTotal1:
			return ScreenIDNoChange, isdata.UpdateResetTotal1{}, true
		case TotalScreenIndexTotal2:
			return ScreenIDNoChange, isdata.UpdateResetTotal2{}, true
			//case TotalScreenIndexLifetime:
			//	return ScreenIDNoChange, isdata.UpdateResetLifetime{}, true
		}
	case isdata.KeyUp, isdata.KeyUpHold, isdata.KeyDown, isdata.KeyDownHold, isdata.KeyRight, isdata.KeyRightHold, isdata.KeyLeft, isdata.KeyLeftHold, isdata.KeyEnter, isdata.KeyEnterHold:
		return s.menu.Key(key)
	}

	return ScreenIDNoChange, nil, true
}
