package isdata

// LcdPixel is used to represent a LCD pixel
type LcdPixel struct {
	X int  `json:"x"`
	Y int  `json:"y"`
	V bool `json:"v"`
}

// LcdBlt is used to define a block of data in the LCD
type LcdBlt struct {
	X    int   `json:"x"`
	Y    int   `json:"y"`
	W    int   `json:"w"`
	H    int   `json:"h"`
	Data []int `json:"data"`
}

// MakeBltBlock returns a Blt of one solid color
func MakeBltBlock(x, y, w, h, v int) LcdBlt {
	data := make([]int, w*h)
	if v != 0 {
		for i := range data {
			data[i] = v
		}
	}
	return LcdBlt{
		X:    x,
		Y:    y,
		W:    w,
		H:    h,
		Data: data,
	}
}
