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
	txt := "Hello, World!"
	width := tightpixel15.Font.MeasureString(txt)
	fmt.Println("width: ", width)
	img := image.NewRGBA(image.Rect(0, 0, width, 10))
	tightpixel15.Font.DrawString(img, 0, 0, txt, color.Black)

	f, _ := os.OpenFile("hello.png", os.O_CREATE|os.O_RDWR, 0644)
	png.Encode(f, img)
}
