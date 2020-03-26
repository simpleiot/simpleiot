package isui

import (
	"image/draw"
	"strconv"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// StatusScreen2 is used to display status info
type StatusScreen2 struct {
	softKeys *SoftKeys
	icons    *Icons
	state    *isdata.State
	config   *isdata.Config
}

// NewStatusScreen2 initializes and returns a HomeScreen
func NewStatusScreen2(state *isdata.State, config *isdata.Config) *StatusScreen2 {
	return &StatusScreen2{
		softKeys: NewSoftKeys("home", "field", "prod."),
		icons:    NewIcons(true, false, false),
		state:    state,
		config:   config,
	}
}

// Render updates the home screen, and provides an image
func (s *StatusScreen2) Render(img draw.Image) {
	Clear(img)

	x := 2
	y1, y2, yBreak, y3, y4 := 9, 20, 30, 33, 44

	DrawTxt(img, s.config.FieldConfigs[s.config.CurrentFieldIndex].Description+" - "+s.config.ProductConfigs[s.config.CurrentProductIndex].Description, x, y1, tightpixel15.Font)
	DrawTxt(img, "Total:", x, y2, tightpixel15.Font)
	DrawTxt(img, "Avg. Armed Flow:", x, y3, tightpixel15.Font)
	DrawTxt(img, "Over:", x, y4, tightpixel15.Font)

	x = 94

	// Total
	DrawTxtRight(img, strconv.FormatFloat(s.state.FieldStates[s.config.CurrentFieldIndex][s.config.CurrentProductIndex].Total, 'f', 0, 64), x, y2, tightpixel15.Font)
	DrawTxt(img, "Gallons", x+4, y2, tightpixel15.Font)

	// Break
	Line(img, 1, yBreak, 163, yBreak)

	// Avg Flow
	avgFlowStr := strconv.FormatFloat(s.state.AvgArmedFlowRate, 'f', 0, 64)
	DrawTxtRight(img, avgFlowStr, x, y3, tightpixel15.Font)

	// Avg Over
	timeSinceArm := strconv.FormatFloat(s.state.DurationArmed.Hours(), 'f', 1, 64)
	DrawTxtRight(img, timeSinceArm, x, y4, tightpixel15.Font)
	DrawTxt(img, "hrs", x+4, y4, tightpixel15.Font)

	// icons
	// page indicator
	s.icons.SetPage("page indicator", 2) // set page indicator icon to home
	s.icons.Render(img)

	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *StatusScreen2) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1: // Home
		return ScreenIDHome, nil, true
	case isdata.KeySK2: // Field Menu Screen
		return ScreenIDFieldMenu1, nil, true
	case isdata.KeySK3: // Product Menu Screen
		return ScreenIDProductMenu1, nil, true
	case isdata.KeyLeft:
		return ScreenIDStatus1, nil, true
	case isdata.KeyRight:
		return ScreenIDStatus3, nil, true
	}

	return ScreenIDNoChange, nil, true
}
