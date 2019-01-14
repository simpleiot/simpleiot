package isui

import (
	"bytes"
	"fmt"
	"image"

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

// GetLcdAssetBlt returns a LcdBlt for a particular LCD asset bmp
func GetLcdAssetBlt(x, y int, name string) (isdata.LcdBlt, error) {
	data := lcdassets.Asset("/" + name)

	img, err := bmp.Decode(bytes.NewReader(data))

	if err != nil {
		return isdata.LcdBlt{}, err
	}

	return ImageToBlt(x, y, img), nil
}

// ImageToBlt converts an image to a LCD Blt structure
func ImageToBlt(x, y int, img image.Image) isdata.LcdBlt {
	rect := img.Bounds()
	min, max := rect.Min, rect.Max
	w := max.X - min.X
	h := max.Y - min.Y
	ret := isdata.NewLcdBlt(x, y, w, h, false)
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			r, _, _, _ := img.At(x, y).RGBA()
			if r > 0 {
				ret.Data[y*w+x] = false
			} else {
				ret.Data[y*w+x] = true
			}
		}
	}

	return ret
}
