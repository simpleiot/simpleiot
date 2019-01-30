package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/simpleiot/simpleiot/farmation/isui"
)

func main() {
	fmt.Println("Draw test")
	img := image.NewRGBA(image.Rect(0, 0, 128, 64))

	isui.Rect(img, 10, 10, 50, 50, color.Black)

	isui.DrawTxt(img, "Hi there", 64, 10)

	err := isui.DrawBmp(img, "pump-off.bmp", 64, 32)
	if err != nil {
		fmt.Println("Error drawing bmp")
	}

	f, _ := os.OpenFile("out.png", os.O_CREATE|os.O_RDWR, 0644)
	png.Encode(f, img)
}
