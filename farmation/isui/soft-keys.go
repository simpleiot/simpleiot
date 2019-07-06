package isui

import (
	"fmt"
	"image/draw"
	"log"
	"time"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
)

// SoftKey holds all the info necessary to render a soft key
type SoftKey struct {
	label     string
	blinking  bool
	lastBlink time.Time
	on        bool
}

// SoftKeys is an array of 4 SoftKey types
type SoftKeys [4]SoftKey

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

// SetLabel set a menu label
func (k *SoftKeys) SetLabel(index int, label string) {
	if index >= len(k) {
		log.Println("SoftKeys index out of range: ", index)
		return
	}

	k[index].label = label
}

// SetBlinking sets blinking of soft key at index
func (k *SoftKeys) SetBlinking(index int, blinking bool) {
	//if !k[index].blinking {
	//		k[index].lastBlink = time.Now()
	//	}
	k[index].blinking = blinking
}

// location of menu strings:
// 2,53
// 35,53
// 67,53

var labelXOffsets = []int{15, 47, 80, 111}
var menuItemWidth = 27

// Render renders the menu section of the screen
func (k *SoftKeys) Render(img draw.Image, x, y int) {
	if len(k[0].label) > 0 {
		Polyline(img,
			0, 55,
			1, 54,
			29, 54,
			31, 56,
			31, 63)
	}

	if len(k[1].label) > 0 {
		Polyline(img,
			31, 56,
			33, 54,
			61, 54,
			63, 56,
			63, 63)
	}

	if len(k[2].label) > 0 {
		Polyline(img,
			63, 56,
			65, 54,
			94, 54,
			96, 56,
			96, 63)
	}

	if len(k[3].label) > 0 {
		Polyline(img,
			96, 56,
			98, 54,
			126, 54,
			127, 55)
	}

	for i, key := range k {
		if key.blinking {
			if time.Since(key.lastBlink) >= 100*time.Millisecond {
				k[i].lastBlink = time.Now()
				k[i].on = !key.on
			}
			fmt.Println(key.on)
			if key.on {
				drawKey(img, key.label, i, x, y)
			}
		} else {
			drawKey(img, key.label, i, x, y)
		}
	}
}

func drawKey(img draw.Image, label string, index, x, y int) {
	if label != "" {
		if allCaps(label) { // if label is all caps, move down two pixels to center
			DrawTxtCentered(img, label,
				labelXOffsets[index]+x, y+2,
				tightpixel15.Font)
		} else {
			DrawTxtCentered(img, label,
				labelXOffsets[index]+x, y,
				tightpixel15.Font)
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
