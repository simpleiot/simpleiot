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
	height := tightpixel15.Font.GetHeight()
	fmt.Println("width: ", width)
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	widthActual := tightpixel15.Font.DrawString(img, 0, 0, txt, color.Black)
	imgS := img.SubImage(image.Rectangle{
		image.Point{0, 0},
		image.Point{widthActual, height},
	})
	fmt.Println("widthActual: ", widthActual)

	f, _ := os.OpenFile("hello.png", os.O_CREATE|os.O_RDWR, 0644)
	png.Encode(f, imgS)
}
