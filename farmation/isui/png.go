package isui

import (
	"image"
	"image/draw"
	"image/png"
)

// DrawPng draws a bmp on the image
func DrawPng(img draw.Image, name string, x, y int) error {
	imgPng, err := png.Decode(GetLcdAsset(name))
	if err != nil {
		return err
	}

	dp := image.Point{x, y}
	sr := imgPng.Bounds()
	r := sr.Add(dp)

	draw.Draw(img, r, imgPng, sr.Min, draw.Over)

	return nil
}
