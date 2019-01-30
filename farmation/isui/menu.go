package isui

import (
	"image/draw"
	"log"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
)

// Menu is used to store menu state and render the menu
type Menu struct {
	labels [4]string
}

// SetLabel set a menu label
func (m *Menu) SetLabel(index int, label string) {
	if index >= len(m.labels) {
		log.Println("Menu index out of range: ", index)
		return
	}

	m.labels[index] = label
}

// location of menu strings:
// 2,53
// 35,53
// 67,53

var labelXOffsets = []int{2, 35, 67, 80}
var menuItemWidth = 27

// Render renders the menu section of the screen
func (m *Menu) Render(img draw.Image, x, y int) (err error) {
	err = DrawBmp(img, "sk-lines.bmp", x, y)

	if err != nil {
		log.Println("Error loading sk-lines")
	}

	for i, l := range m.labels {
		if l != "" {
			DrawTxt(img, l,
				labelXOffsets[i]+x, y,
				tightpixel15.Font)
		}
	}

	return
}
