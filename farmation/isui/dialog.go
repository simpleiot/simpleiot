package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// Dialog represents a UI widget that displays a dialog on the screen
type Dialog struct {
	visible  bool
	current  isdata.DialogMsg
	callback func(yes bool)
	queue    chan isdata.DialogMsg
}

// NewDialog creates a new dialog
func NewDialog() *Dialog {
	return &Dialog{
		queue: make(chan isdata.DialogMsg, 10),
	}
}

// Render is used to draw the dialog
func (d *Dialog) Render(img draw.Image) {
	if !d.visible {
		return
	}

	ClearRect(img, 20, 20, 108, 44)
	Rect(img, 20, 20, 108, 44)
}

// Key handles key input
func (d *Dialog) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	if d.visible {
		return ScreenIDNoChange, nil, true
	}
	return ScreenIDNoChange, nil, false
}
