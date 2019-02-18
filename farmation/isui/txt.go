package isui

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/pbnjay/pixfont"
	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// DrawTxt draws text into an image
func DrawTxt(img draw.Image, txt string, x, y int, font *pixfont.PixFont) {
	font.DrawString(img, x, y, txt, color.Black)
}

// DrawTxtCentered draws text into an image centered around x,y
func DrawTxtCentered(img draw.Image, txt string, x, y int, font *pixfont.PixFont) {
	length := font.MeasureString(txt)
	x -= length / 2
	font.DrawString(img, x, y, txt, color.Black)
}

// DrawTxtRight draws right justified text with x,y at end of string
func DrawTxtRight(img draw.Image, txt string, x, y int, font *pixfont.PixFont) {
	length := font.MeasureString(txt)
	x -= length
	font.DrawString(img, x, y, txt, color.Black)
}

// Heading draws the screen heading with a box around it
func Heading(img draw.Image, txt string) {
	DrawTxtCentered(img, txt, 64, 2, tightpixel15.Font)
	width := tightpixel15.Font.MeasureString(txt)
	Rect(img, 64-width/2-2, 0, width+2, 11)
}

// RenderTxt returns an image of rendered text
func RenderTxt(txt string) image.Image {
	width := tightpixel15.Font.MeasureString(txt)
	height := tightpixel15.Font.GetHeight()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	tightpixel15.Font.DrawString(img, 0, 0, txt, color.Black)
	//imgCropped := img.SubImage(image.Rect(0, 0, widthActual, height))

	//f, _ := os.OpenFile("RenderTxt.png", os.O_CREATE|os.O_RDWR, 0644)
	//png.Encode(f, img)
	return img
}

// RenderTxtBlt renders text into a Blt
func RenderTxtBlt(x, y int, txt string) isdata.LcdBlt {
	img := RenderTxt(txt)
	return ImageToBltA(x, y, img, true)
}
