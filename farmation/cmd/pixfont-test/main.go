package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
)

func main() {
	txt := "ai"
	width := tightpixel15.Font.MeasureString(txt)
	fmt.Println("width: ", width)
	img := image.NewRGBA(image.Rect(0, 0, width, 10))
	widthActual := tightpixel15.Font.DrawString(img, 0, 0, txt, color.Black)
	imgS := img.SubImage(image.Rectangle{
		image.Point{0, 0},
		image.Point{widthActual, 10},
	})
	fmt.Println("widthActual: ", widthActual)

	f, _ := os.OpenFile("hello.png", os.O_CREATE|os.O_RDWR, 0644)
	png.Encode(f, imgS)
}
