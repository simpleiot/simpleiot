package isui

import "image/draw"

// Arrow is used to draw an arrow on image
func Arrow(img draw.Image, x, y int) {
	width := 6
	height := 4
	Line(img, x, y, x+width, y)
	Line(img, x+width, y, x+width-height/2, y+height/2)
	Line(img, x+width, y, x+width-height/2, y-height/2)
}
