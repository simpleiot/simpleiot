package isui

import (
	"image/draw"
	"log"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
)

// SoftKeys is used to store menu state and render the menu
type SoftKeys struct {
	labels [4]string
}

// SetLabel set a menu label
func (m *SoftKeys) SetLabel(index int, label string) {
	if index >= len(m.labels) {
		log.Println("SoftKeys index out of range: ", index)
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
func (m *SoftKeys) Render(img draw.Image, x, y int) (err error) {
	if len(m.labels[0]) > 0 {
		Polyline(img,
			0, 55,
			1, 54,
			29, 54,
			31, 56,
			31, 63)
	}

	if len(m.labels[1]) > 0 {
		Polyline(img,
			31, 56,
			33, 54,
			61, 54,
			63, 56,
			63, 63)
	}

	if len(m.labels[2]) > 0 {
		Polyline(img,
			63, 56,
			65, 54,
			94, 54,
			96, 56,
			96, 63)
	}

	if len(m.labels[3]) > 0 {
		Polyline(img,
			96, 56,
			98, 54,
			126, 54,
			127, 55)
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
