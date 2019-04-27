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
	arrowPos int
	menu     Menu
}

// NewTotalsScreen gives new Totals screen to screen.go
func NewTotalsScreen(state *isdata.State, config *isdata.Config) *TotalsScreen {
	isdata.InitState(state) // make sure that ProductStates and FieldStates arrays are large enough
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
	s.menu.ResetItems()
	s.menu.AddItemInt("Current Field",
		s.state.FieldStates[s.config.CurrentFieldIndex].Total)
	s.menu.AddItemFloat("Total 1", s.state.Total1)
	s.menu.AddItemFloat("Total 2", s.state.Total2)
	s.menu.AddItemInt("Product 1 Total", s.state.ProductStates[0].Total)
	s.menu.AddItemInt("Product 2 Total", s.state.ProductStates[1].Total)
	s.menu.AddItemInt("Product 3 Total", s.state.ProductStates[2].Total)
	s.menu.AddItemInt("Product 4 Total", s.state.ProductStates[3].Total)
	s.menu.AddItemInt("Product 5 Total", s.state.ProductStates[4].Total)
	s.menu.AddItemInt("Lifetime Total", s.state.LifetimeTotal)

	Heading(img, "Totals")
	s.menu.Render(img)
	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *TotalsScreen) Key(key isdata.Key) (ScreenID, interface{}) {
	switch key {
	case isdata.KeySK1:
		s.menu.ResetArrowPos() // return arrow to top of screen
		return ScreenIDMainMenu, nil
	case isdata.KeyUp, isdata.KeyDown, isdata.KeyRight, isdata.KeyLeft, isdata.KeyEnter:
		return s.menu.Key(key)
	}

	return ScreenIDNoChange, nil
}
