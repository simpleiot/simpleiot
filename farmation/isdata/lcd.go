package isdata

// LcdPixel is used to represent a LCD pixel
type LcdPixel struct {
	X int  `json:"x"`
	Y int  `json:"y"`
	V bool `json:"v"`
}
