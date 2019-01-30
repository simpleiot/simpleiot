package isui

import (
	"image/draw"
	"strconv"

	"github.com/simpleiot/simpleiot/farmation/fonts/agencyfbbold20"
	"github.com/simpleiot/simpleiot/farmation/fonts/agencyfbbold40"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// HomeScreen is used to render the home screen
type HomeScreen struct {
	menu   *Menu
	state  *isdata.State
	config *isdata.Config
}

// NewHomeScreen initializes and returns a HomeScreen
func NewHomeScreen(state *isdata.State, config *isdata.Config) *HomeScreen {
	menu := Menu{}
	menu.SetLabel(0, "menu")
	menu.SetLabel(1, "mode")
	menu.SetLabel(2, "pump")

	return &HomeScreen{
		menu:   &menu,
		state:  state,
		config: config,
	}
}

// Render updates the home screen, and provides an image
func (s *HomeScreen) Render(img draw.Image) {
	Clear(img)
	DrawTxt(img, strconv.Itoa(int(s.state.FlowRate)), 4, 12, agencyfbbold40.Font)
	DrawTxt(img, "963", 67, 11, agencyfbbold20.Font)
	DrawTxt(img, "963", 67, 29, agencyfbbold20.Font)

	s.menu.Render(img, 0, 54)
}
