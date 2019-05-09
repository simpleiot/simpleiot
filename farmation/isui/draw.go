package isui

import (
	"image"
	"image/color"
	"image/draw"
	"log"

	"github.com/StephaneBunel/bresenham"
)

// Rect draws a rectangle
func Rect(img draw.Image, x, y, w, h int) {
	Polyline(img,
		x, y,
		x+w, y,
		x+w, y+h,
		x, y+h,
		x, y)
}

// RectFilled draws a filled rectangle
func RectFilled(img draw.Image, x, y, w, h int) {
	for xI := x; xI < x+w; xI++ {
		Line(img, xI, y, xI, y+h)
	}
}

// Clear clears an image to white color
func Clear(img draw.Image) {
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.ZP, draw.Over)
}

// ClearRect clears a portion of the screen
func ClearRect(img draw.Image, x, y, w, h int) {
	min := image.Point{x, y}
	max := image.Point{x + w, y + h}
	bounds := image.Rectangle{min, max}
	draw.Draw(img, bounds, &image.Uniform{color.White}, image.ZP, draw.Over)
}

// Line draw a line between two points
func Line(img draw.Image, x1, y1, x2, y2 int) {
	bresenham.Bresenham(img, x1, y1, x2, y2, color.Black)
}

// LineWhite draw a white line between two points
func LineWhite(img draw.Image, x1, y1, x2, y2 int) {
	bresenham.Bresenham(img, x1, y1, x2, y2, color.White)
}

// Polyline draws a multipoint line
func Polyline(img draw.Image, p ...int) {
	if len(p) < 4 {
		log.Println("Error, Polyline requires at least 4 points")
	}
	for i := 0; i < len(p)-2; i += 2 {
		Line(img, p[i], p[i+1], p[i+2], p[i+3])
	}
}
