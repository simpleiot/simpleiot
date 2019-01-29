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
	fmt.Println("MeasureString returned: ", width)
	height := tightpixel15.Font.GetHeight()
	fmt.Println("width: ", width)
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	widthActual := tightpixel15.Font.DrawString(img, 0, 0, txt, color.Black)
	fmt.Println("DrawString returned: ", widthActual)
	imgS := img.SubImage(image.Rect(0, 0, widthActual, height))
	fmt.Println("widthActual: ", widthActual)

	f, _ := os.OpenFile("hello.png", os.O_CREATE|os.O_RDWR, 0644)
	png.Encode(f, imgS)
}
