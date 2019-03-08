package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// TotalsScreen
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
	menu.AddItemInt("Current Field",
		state.FieldStates[config.CurrentFieldIndex].Total)
	menu.AddItemInt("Total 1", state.Total1)
	menu.AddItemInt("Total 2", state.Total2)
	menu.AddItemInt("Product 1 Total", state.ProductStates[0].Total)
	menu.AddItemInt("Product 2 Total", state.ProductStates[1].Total)
	menu.AddItemInt("Product 3 Total", state.ProductStates[2].Total)
	menu.AddItemInt("Product 4 Total", state.ProductStates[3].Total)
	menu.AddItemInt("Product 5 Total", state.ProductStates[4].Total)
	menu.AddItemInt("Lifetime Total", state.LifetimeTotal)

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
	s.menu.Render(img)
	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *TotalsScreen) Key(key isdata.Key) (ScreenID, interface{}) {
	switch key {
	case isdata.KeySK1:
		return ScreenIDMainMenu, nil
	case isdata.KeyUp, isdata.KeyDown, isdata.KeyRight, isdata.KeyLeft, isdata.KeyEnter:
		return s.menu.Key(key)
	}

	return ScreenIDNoChange, nil
}
