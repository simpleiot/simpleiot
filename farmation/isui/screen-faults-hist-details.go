package isui

import (
	"fmt"
	"image/draw"
	"io"
	"strconv"
	"time"

	"github.com/pbnjay/pixfont"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// FaultsHistDetailsScreen is used to display details for a fault in history
type FaultsHistDetailsScreen struct {
	softKeys *SoftKeys
	state    *isdata.State
	config   *isdata.Config
	fault    data.Sample
}

// NewFaultsHistDetailsScreen initializes and returns a screen
func NewFaultsHistDetailsScreen(state *isdata.State, config *isdata.Config) *FaultsHistDetailsScreen {

	return &FaultsHistDetailsScreen{
		softKeys: NewSoftKeys("back"),
		state:    state,
		config:   config,
	}
}

// Render updates the home screen, and provides an image
func (s *FaultsHistDetailsScreen) Render(img draw.Image) {
	Clear(img)
	Heading(img, isdata.SampleTypeToDispVerbose(s.fault.Type))

	// display fault time
	x := 1
	font := tightpixel15.Font
	t := s.fault.Time
	hour, min, sec := Clock(t, true)
	clockTime := hour + ":" + min + ":" + sec
	year, month, day := Date(t, true, false)
	date := year + "/" + month + "/" + day

	weekDay := t.Weekday()
	var weekDayStr string
	switch weekDay {
	case time.Wednesday:
		weekDayStr = "Wed."
	case time.Thursday:
		weekDayStr = "Thurs."
	case time.Saturday:
		weekDayStr = "Sat."
	default:
		weekDayStr = weekDay.String()
	}

	DrawTxt(img, weekDayStr+" "+date+", "+clockTime, x, 15, font)

	// display value that triggered fault
	y := 28
	switch s.fault.Type {
	case isdata.SampleTypeFaultFlowOff:
		DrawTxt(img, "Flow: "+strconv.FormatFloat(s.fault.Value, 'f', 0, 64), x, y, font)
	case isdata.SampleTypeFaultPresLow:
		pressureStr := "Pressure: " + strconv.FormatFloat(s.fault.Value, 'f', 0, 64)
		if s.fault.Attributes["shutdownThreshold"] != 0 {
			pressureStr = pressureStr + ", Threshold: " + strconv.FormatFloat(s.fault.Attributes["shutdownThreshold"], 'f', 0, 64)
		}
		DrawTxt(img, pressureStr, x, y, font)
	}

	// display input states as icon "-" On, Off, NA
	x = 20
	y = 43
	DrawTxtCentered(img, "-", x, y, font)
	DrawTxtCentered(img, "-", 3*x+2, y, font)
	DrawTxtCentered(img, "-", 5*x+6, y, font)
	y = 41
	drawAndErr(img, "injector.png", x-17, y, font)
	drawAndErr(img, "water-on.png", 3*x-12, y-1, font)
	drawAndErr(img, "irrigator.png", 5*x-13, y+1, font)
	x += 5
	y = 43
	s.drawInState(img, "inputInjector", x, y, font)
	x += 42
	s.drawInState(img, "inputWaterOn", x, y, font)
	x += 45
	s.drawInState(img, "inputIrrigator", x, y, font)

	s.softKeys.Render(img, 0, 54)
}

func drawAndErr(img draw.Image, file string, x, y int, font *pixfont.PixFont) {
	err := DrawPng(img, file, x, y)
	if err != nil && err != io.ErrUnexpectedEOF {
		s := fmt.Sprintf("error drawing %s: %s", file, err)
		fmt.Println(s)
	}
}

func (s *FaultsHistDetailsScreen) drawInState(img draw.Image, key string, x, y int, font *pixfont.PixFont) {
	switch s.fault.Attributes[key] {
	case float64(isdata.InputStateNA):
		DrawTxt(img, "NA", x, y, font)
	case float64(isdata.InputStateOff):
		DrawTxt(img, "Off", x, y, font)
	case float64(isdata.InputStateOn):
		DrawTxt(img, "On", x, y, font)
	}
}
