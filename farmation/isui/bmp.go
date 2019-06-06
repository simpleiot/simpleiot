package isui

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"io"

	"github.com/simpleiot/simpleiot/farmation/assets/lcdassets"
	"github.com/simpleiot/simpleiot/farmation/isdata"
	"golang.org/x/image/bmp"
)

func bmpTest() {
	data := lcdassets.Asset("/splash.bmp")

	img, err := bmp.Decode(bytes.NewReader(data))

	if err != nil {
		fmt.Println("Error decoding image: ", err)
		return
	}

	fmt.Println("color.Model: ", img.ColorModel())
	fmt.Println("Bounds: ", img.Bounds())
	fmt.Println("At(0,0): ", img.At(0, 0))
}

// DrawBmp draws a bmp on the image
func DrawBmp(img draw.Image, name string, x, y int) error {
	imgBmp, err := bmp.Decode(GetLcdAsset(name))

	if err != nil {
		return err
	}

	dp := image.Point{x, y}
	sr := imgBmp.Bounds()
	r := sr.Add(dp)

	draw.Draw(img, r, imgBmp, sr.Min, draw.Over)

	return nil
}

// GetLcdAsset returns an io.Reader
// Used by bmp and png functions
func GetLcdAsset(name string) io.Reader {
	data := lcdassets.Asset("/" + name)

	return bytes.NewReader(data)
}

// ImageToBlt converts an image to a LCD Blt structure
func ImageToBlt(x, y int, img image.Image, invert bool) isdata.LcdBlt {
	rect := img.Bounds()
	min, max := rect.Min, rect.Max
	w := max.X - min.X
	h := max.Y - min.Y
	ret := isdata.NewLcdBlt(x, y, w, h, false)
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			r, _, _, _ := img.At(x, y).RGBA()
			if r > 0 {
				ret.Data[y*w+x] = invert
			} else {
				ret.Data[y*w+x] = !invert
			}
		}
	}

	return ret
}

// ImageToBltA converts an image to a LCD Blt structure
func ImageToBltA(x, y int, img image.Image, invert bool) isdata.LcdBlt {
	rect := img.Bounds()
	min, max := rect.Min, rect.Max
	w := max.X - min.X
	h := max.Y - min.Y
	ret := isdata.NewLcdBlt(x, y, w, h, false)
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a > 0 {
				ret.Data[y*w+x] = invert
			} else {
				ret.Data[y*w+x] = !invert
			}
		}
	}

	return ret
}
