package isui

import (
	"image/draw"
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
	DrawTxt(img, "hi there", 10, 10)
	s.menu.Render(img, 0, 54)
}
