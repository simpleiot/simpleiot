package isdata

// LcdSetPixel is message used to set a LCD pixel
type LcdSetPixel struct {
	X int  `json:"x"`
	Y int  `json:"y"`
	V bool `json:"v"`
}
