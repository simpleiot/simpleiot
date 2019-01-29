package isui

import (
	"image"
	"image/draw"
	"log"
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
}

// location of menu strings:
// 2,53
// 35,53
// 67,53

var labelXOffsets = []int{2, 53, 67, 80}
var menuItemWidth = 27

// Render renders the menu section of the screen
func (m *Menu) Render() image.Image {
	img, err := GetLcdAsset("sk-lines.bmp")
	if err != nil {
		log.Println("Error loading sk-lines")
	}

	size := img.Bounds().Size()

	for i, l := range m.labels {
		if l != "" {
			lImg := RenderTxt(l)
			r := image.Rect(labelXOffsets[i], 0, labelXOffsets[i]+27, size.Y)
			draw.Draw(img, r, lImg, image.Point{}, image.Over)

		}
	}

	return img
}
