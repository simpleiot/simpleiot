package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

type dialogMsg struct {
	Text     string
	Ok       bool
	Yes      bool
	No       bool
	Callback func(yes bool)
}

// Dialog represents a UI widget that displays a dialog on the screen
type Dialog struct {
	visible  bool
	current  dialogMsg
	callback func(yes bool)
	queue    chan dialogMsg
}

// NewDialog creates a new dialog
func NewDialog() *Dialog {
	return &Dialog{
		queue: make(chan dialogMsg, 10),
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
