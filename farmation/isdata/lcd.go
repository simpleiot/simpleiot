package isdata

import "log"

// LcdPixel is used to represent a LCD pixel
type LcdPixel struct {
	X int  `json:"x"`
	Y int  `json:"y"`
	V bool `json:"v"`
}

// LcdBlt is used to define a block of data in the LCD
type LcdBlt struct {
	X    int    `json:"x"`
	Y    int    `json:"y"`
	W    int    `json:"w"`
	H    int    `json:"h"`
	Data []bool `json:"data"`
}

// NewLcdBlt creates a blt structure and allocates data slice.
func NewLcdBlt(x, y, w, h int, v bool) LcdBlt {
	ret := LcdBlt{
		X:    x,
		Y:    y,
		W:    w,
		H:    h,
		Data: make([]bool, w*h),
	}

	if v {
		for i := range ret.Data {
			ret.Data[i] = v
		}
	}

	return ret
}

// At returns the value at a specific location
func (b *LcdBlt) At(x, y int) bool {
	i := b.W*y + x
	if i >= len(b.Data) {
		log.Println("Warning, data out of range for LcdBlt.At()")
		return false
	}

	return b.Data[i]
}

// Set is used to set a Blt value
func (b *LcdBlt) Set(x, y int, v bool) {
	i := b.W*y + x
	if i >= len(b.Data) {
		log.Println("Warning, data out of range for LcdBlt.Set()")
		return
	}

	b.Data[i] = v
}

// Update updates a portion of a LcdBlt with another LcdBlt. This allows a
// LcdBlt to be used as a framebuffer, or to compose blts.
func (b *LcdBlt) Update(blt LcdBlt) {
	for y := 0; y < blt.H; y++ {
		for x := 0; x < blt.W; x++ {
			b.Set(blt.X+x, blt.Y+y, blt.At(x, y))
		}
	}
}

// UpdateSolid is used to update a region in a Blt with a solid color.
func (b *LcdBlt) UpdateSolid(blt LcdBltSolid) {
	for y := 0; y < blt.H; y++ {
		for x := 0; x < blt.W; x++ {
			b.Set(blt.X+x, blt.Y+y, blt.V)
		}
	}
}

// LcdBltSolid is used to define a block of one color
type LcdBltSolid struct {
	X int  `json:"x"`
	Y int  `json:"y"`
	W int  `json:"w"`
	H int  `json:"h"`
	V bool `json:"v"`
}
