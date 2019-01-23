package isui

import (
	"image"
	"image/color"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// RenderTxt renders text into a Blt
func RenderTxt(x, y int, txt string) isdata.LcdBlt {
	width := tightpixel15.Font.MeasureString(txt)
	img := image.NewRGBA(image.Rect(0, 0, width, 10))
	tightpixel15.Font.DrawString(img, 0, 0, txt, color.Black)
	//f, _ := os.OpenFile("RenderTxt.png", os.O_CREATE|os.O_RDWR, 0644)
	//png.Encode(f, img)
	return ImageToBltA(x, y, img, true)
}
