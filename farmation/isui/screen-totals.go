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
	isdata.InitState(state) // make sure that FieldStates[s.config.CurrentFieldIndex] and FieldStates arrays are large enough
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

	Heading(img, "Totals")

	s.menu.ResetItems()
	s.menu.AddItemFloat(s.config.FieldConfigs[s.config.CurrentFieldIndex].Description,
		s.state.FieldStates[s.config.CurrentFieldIndex][s.config.CurrentProductIndex].Total)
	s.menu.AddItemFloat("Total 1", s.state.Total1)
	s.menu.AddItemFloat("Total 2", s.state.Total2)
	s.menu.AddItemFloat(s.config.ProductConfigs[0].Description+" Total", s.state.FieldStates[s.config.CurrentFieldIndex][0].Total)
	s.menu.AddItemFloat(s.config.ProductConfigs[1].Description+" Total", s.state.FieldStates[s.config.CurrentFieldIndex][1].Total)
	s.menu.AddItemFloat(s.config.ProductConfigs[2].Description+" Total", s.state.FieldStates[s.config.CurrentFieldIndex][2].Total)
	s.menu.AddItemFloat(s.config.ProductConfigs[3].Description+" Total", s.state.FieldStates[s.config.CurrentFieldIndex][3].Total)
	s.menu.AddItemFloat(s.config.ProductConfigs[4].Description+" Total", s.state.FieldStates[s.config.CurrentFieldIndex][4].Total)
	s.menu.AddItemFloat("Lifetime Total", s.state.LifetimeTotal)

	s.menu.Render(img)
	s.softKeys.Render(img, 0, 54)
}

// define position of various things on total screen
const (
	TotalScreenIndexCurrentField int = iota
	TotalScreenIndexTotal1
	TotalScreenIndexTotal2
	TotalScreenProduct1
	TotalScreenProduct2
	TotalScreenProduct3
	TotalScreenProduct4
	TotalScreenProduct5
	TotalScreenLifetime
)

// Key processes keypad input to this screen
func (s *TotalsScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1: // Back
		s.menu.ResetArrowPos() // return arrow to top of screen
		return ScreenIDPrev, nil, true
	case isdata.KeySK2: // Reset
		switch s.menu.GetArrowPos() {
		case TotalScreenIndexTotal1:
			return ScreenIDNoChange, isdata.UpdateResetTotal1{}, true
		case TotalScreenIndexTotal2:
			return ScreenIDNoChange, isdata.UpdateResetTotal2{}, true
			//case TotalScreenLifetime:
			//	return ScreenIDNoChange, isdata.UpdateResetLifetime{}, true
		case TotalScreenProduct1:
			return ScreenIDNoChange, isdata.UpdateResetProduct1{}, true
		case TotalScreenProduct2:
			return ScreenIDNoChange, isdata.UpdateResetProduct2{}, true
		case TotalScreenProduct3:
			return ScreenIDNoChange, isdata.UpdateResetProduct3{}, true
		case TotalScreenProduct4:
			return ScreenIDNoChange, isdata.UpdateResetProduct4{}, true
		case TotalScreenProduct5:
			return ScreenIDNoChange, isdata.UpdateResetProduct5{}, true
		}
	case isdata.KeyUp, isdata.KeyDown, isdata.KeyRight, isdata.KeyLeft, isdata.KeyEnter:
		return s.menu.Key(key)
	}

	return ScreenIDNoChange, nil, true
}
