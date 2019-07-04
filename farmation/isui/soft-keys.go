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

// NewSoftKeys creates soft keys with given items
func NewSoftKeys(items ...string) *SoftKeys {
	if len(items) > 4 {
		log.Println("Error, can only have 4 soft keys")
		return &SoftKeys{}
	}

	ret := SoftKeys{}

	for i, key := range items {
		ret.SetLabel(i, key)
	}

	return &ret
}

// location of menu strings:
// 2,53
// 35,53
// 67,53

var labelXOffsets = []int{15, 47, 80, 111}
var menuItemWidth = 27

// Render renders the menu section of the screen
func (m *SoftKeys) Render(img draw.Image, x, y int) {
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
			if allCaps(l) { // if label is all caps, move down two pixels to center
				DrawTxtCentered(img, l,
					labelXOffsets[i]+x, y+2,
					tightpixel15.Font)
			} else {
				DrawTxtCentered(img, l,
					labelXOffsets[i]+x, y,
					tightpixel15.Font)
			}
		}
	}
}

func allCaps(s string) bool {
	match := false
	for _, char := range s {
		for _, letter := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
			if char == letter {
				match = true
			}
		}
		if !match {
			return false
		}
	}
	return true
}
