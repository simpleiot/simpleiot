package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/simpleiot/simpleiot/farmation/fonts/agencyfbbold20"
)

func main() {
	font := agencyfbbold20.Font
	txt := "123"
	width := font.MeasureString(txt)
	fmt.Println("MeasureString returned: ", width)
	height := font.GetHeight()
	fmt.Println("Height: ", height)
	fmt.Println("width: ", width)
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	widthActual := font.DrawString(img, 0, 0, txt, color.Black)
	fmt.Println("DrawString returned: ", widthActual)
	imgS := img.SubImage(image.Rect(0, 0, widthActual, height))
	fmt.Println("widthActual: ", widthActual)

	f, _ := os.OpenFile("hello.png", os.O_CREATE|os.O_RDWR, 0644)
	png.Encode(f, imgS)
}
