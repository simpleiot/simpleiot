package isui

import "image/draw"

// Screen is an interface that generally represents a screen
type Screen interface {
	Render(img draw.Image)
}
