package isui

import (
	"image/color"
	"image/draw"

	"github.com/pbnjay/pixfont"
	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
)

// DrawTxt draws text into an image
func DrawTxt(img draw.Image, txt string, x, y int, font *pixfont.PixFont) {
	font.DrawString(img, x, y, txt, color.Black)
}

// DrawTxtRev draws text with white on black background
func DrawTxtRev(img draw.Image, txt string, x, y int, font *pixfont.PixFont) {
	w := font.MeasureString(txt)
	h := font.GetHeight()
	RectFilled(img, x, y, w, h)
	font.DrawString(img, x, y, txt, color.White)
}

// DrawTxtCentered draws text into an image centered around x,y
// returns the starting x location of the string
func DrawTxtCentered(img draw.Image, txt string, x, y int, font *pixfont.PixFont) int {
	length := font.MeasureString(txt)
	x -= length / 2
	font.DrawString(img, x, y, txt, color.Black)
	return x
}

// DrawTxtRight draws right justified text with x,y at end of string
func DrawTxtRight(img draw.Image, txt string, x, y int, font *pixfont.PixFont) {
	length := font.MeasureString(txt)
	x -= length
	font.DrawString(img, x, y, txt, color.Black)
}

// Heading draws the screen heading with a box around it
func Heading(img draw.Image, txt string) {
	font := tightpixel15.Font
	DrawTxtCentered(img, txt, 64, 2, font)
	width := font.MeasureString(txt)
	Rect(img, 64-width/2-2, 0, width+2, 11)
}
