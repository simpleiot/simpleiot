package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// Widget is an interface that can represent a screen, part of a screen, etc.
type Widget interface {
	Render(img draw.Image)
	// Key can return a new ScreenID if we need to change screens, and
	// a type that describes actions that need to be taken, and a bool
	// that indicates if the Key has handled.
	Key(isdata.Key) (ScreenID, interface{}, bool)
}

// Widgets is a collection of widgets that implements the widget interface. They are designed
// to be stacked. The ones later in the list get rendered last, so the have the highest priority.
type Widgets []Widget

// Add a widget
func (w *Widgets) Add(widget Widget) {
	*w = append(*w, widget)
}

// Render displays the stack of widgets
func (w *Widgets) Render(img draw.Image) {
	for _, widget := range *w {
		(widget).Render(img)
	}
}

// Key handles keycodes in reverse order. If a widget responds with the key is handled, then
// the the Key() method is not called on the rest of the widgets.
func (w *Widgets) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	if key == isdata.KeyPump {
		return ScreenIDPumpMode, nil, true
	}

	for i := len(*w) - 1; i >= 0; i-- {
		screenID, action, handled := (*w)[i].Key(key)
		if handled {
			return screenID, action, handled
		}
	}

	return ScreenIDNoChange, nil, false
}
