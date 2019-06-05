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

// DrawTxtHighlight draws text into an image
// highlights parameter char for cursor
func DrawTxtHighlight(img draw.Image, txt string, highlightChar rune, caps bool, x, y int, font *pixfont.PixFont) {
	spacing := 2
	for _, c := range txt {
		_, w := font.DrawRune(img, x, y, c, color.Black) // draw character
		if c == highlightChar {                          // if this is the highlighted character
			_, w := font.MeasureRune(c)
			h := font.GetHeight()
			if caps {
				RectFilled(img, x-1, y-1, w+2, h-1)
			} else {
				RectFilled(img, x-1, y, w+2, h)
			}
			font.DrawRune(img, x, y, c, color.White)
		}
		x += w + spacing
	}
}

// DrawTxtRev draws text with white on black background (which fits lowercase)
func DrawTxtRev(img draw.Image, txt string, x, y int, font *pixfont.PixFont) {
	w := font.MeasureString(txt)
	h := font.GetHeight()
	RectFilled(img, x-1, y, w+1, h)
	font.DrawString(img, x, y, txt, color.White)
}

// DrawTxtRevLarge draws text with white on black background (which fits uppercase)
func DrawTxtRevLarge(img draw.Image, txt string, x, y int, font *pixfont.PixFont) {
	w := font.MeasureString(txt)
	h := font.GetHeight()
	RectFilled(img, x-1, y-1, w+1, h-1)
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
