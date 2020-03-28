package isui

import (
	"image/draw"
	"strconv"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// GpsScreen ethernet screen
type GpsScreen struct {
	softKeys *SoftKeys
	state    *isdata.State
	config   *isdata.Config
}

// NewGpsScreen creates a gps information screen
func NewGpsScreen(state *isdata.State, config *isdata.Config) *GpsScreen {
	isdata.InitState(state) // make sure that ProductStates and FieldStates arrays are large enough
	return &GpsScreen{
		// update this from sample screen
		softKeys: NewSoftKeys("back"),
		state:    state,
		config:   config,
	}
}

// Render updates the screen, and provides an image
func (s *GpsScreen) Render(img draw.Image) {
	Clear(img)

	Heading(img, "GPS")

	spacing := 11
	x := 10
	y := 10

	if s.state.Location.Fix == "" {
		DrawTxt(img, "No data", 46, 25, tightpixel15.Font)
	} else {

		// typical GPS values:  30.166755,-85.6389933
		// display 7 decimal places
		lat := strconv.FormatFloat(s.state.Location.Lat, 'f', 7, 64)
		long := strconv.FormatFloat(s.state.Location.Long, 'f', 7, 64)
		numSat := strconv.Itoa(int(s.state.Location.NumSat))

		DrawTxt(img, "Num satelites: "+numSat, x, y, tightpixel15.Font)
		DrawTxt(img, "Fix: "+s.state.Location.Fix, x, y+spacing, tightpixel15.Font)
		DrawTxt(img, "Lat: "+lat, x, y+spacing*2, tightpixel15.Font)
		DrawTxt(img, "Long: "+long, x, y+spacing*3, tightpixel15.Font)
	}

	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *GpsScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1Hold: // Back key held -> Home screen
		return ScreenIDHome, nil, true
	case isdata.KeySK1Release: // Back
		return ScreenIDPrev, nil, true
	}

	return ScreenIDNoChange, nil, true
}
