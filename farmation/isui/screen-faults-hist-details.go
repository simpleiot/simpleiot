package isui

import (
	"image/draw"
	"strconv"
	"time"

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
	clockTime := clockTime(t)
	date := date(t, true)

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
		DrawTxt(img, "Pressure: "+strconv.FormatFloat(s.fault.Value, 'f', 0, 64), x, y, font)
	}

	s.softKeys.Render(img, 0, 54)
}
