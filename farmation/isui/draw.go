package isui

import (
	"image"
	"image/color"
	"image/draw"
	"log"

	"github.com/StephaneBunel/bresenham"
)

// HLine draws a horizontal line
func HLine(img draw.Image, x1, y, x2 int, col color.Color) {
	for ; x1 <= x2; x1++ {
		img.Set(x1, y, col)
	}
}

// VLine draws a veritcal line
func VLine(img draw.Image, x, y1, y2 int, col color.Color) {
	for ; y1 <= y2; y1++ {
		img.Set(x, y1, col)
	}
}

// Rect draws a rectangle utilizing HLine() and VLine()
func Rect(img draw.Image, x1, y1, x2, y2 int, col color.Color) {
	HLine(img, x1, y1, x2, col)
	HLine(img, x1, y2, x2, col)
	VLine(img, x1, y1, y2, col)
	VLine(img, x2, y1, y2, col)
}

// Clear clears an image to white color
func Clear(img draw.Image) {
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.ZP, draw.Over)

}

// Line draw a line between two points
func Line(img draw.Image, x1, y1, x2, y2 int) {
	bresenham.Bresenham(img, x1, y1, x2, y2, color.Black)
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
