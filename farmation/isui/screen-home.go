package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/fonts/agencyfbbold20"
	"github.com/simpleiot/simpleiot/farmation/fonts/agencyfbbold40"
)

// HomeScreen is used to render the home screen
type HomeScreen struct {
	menu *Menu
}

// NewHomeScreen initializes and returns a HomeScreen
func NewHomeScreen() *HomeScreen {
	menu := Menu{}
	menu.SetLabel(0, "menu")
	menu.SetLabel(1, "mode")
	menu.SetLabel(2, "pump")

	return &HomeScreen{
		menu: &menu,
	}
}

// Render updates the home screen, and provides an image
func (s *HomeScreen) Render(img draw.Image) {
	Clear(img)
	DrawTxt(img, "963", 4, 12, agencyfbbold40.Font)
	DrawTxt(img, "963", 67, 11, agencyfbbold20.Font)
	DrawTxt(img, "963", 67, 29, agencyfbbold20.Font)

	s.menu.Render(img, 0, 54)
}
