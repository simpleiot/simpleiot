package isui

import (
	"image"
	"image/draw"
	"image/png"
)

// DrawPng draws a bmp on the image
func DrawPng(img draw.Image, name string, x, y int) error {
	a, err := GetLcdAsset(name)

	if a == nil {
		return nil
	}

	if err != nil {
		return err
	}

	imgPng, err := png.Decode(a)
	if err != nil {
		return err
	}

	dp := image.Point{x, y}
	sr := imgPng.Bounds()
	r := sr.Add(dp)

	draw.Draw(img, r, imgPng, sr.Min, draw.Over)

	return nil
}
