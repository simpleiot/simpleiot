package isdata

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

// LcdBltSolid is used to define a block of one color
type LcdBltSolid struct {
	X int  `json:"x"`
	Y int  `json:"y"`
	W int  `json:"w"`
	H int  `json:"h"`
	V bool `json:"v"`
}
