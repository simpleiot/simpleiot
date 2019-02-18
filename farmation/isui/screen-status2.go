package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// StatusScreen2 is used to display status info
type StatusScreen2 struct {
	softKeys *SoftKeys
	state    *isdata.State
	config   *isdata.Config
}

// NewStatusScreen2 initializes and returns a HomeScreen
func NewStatusScreen2(state *isdata.State, config *isdata.Config) *StatusScreen2 {
	softKeys := SoftKeys{}
	softKeys.SetLabel(0, "home")
	softKeys.SetLabel(1, "mode")
	softKeys.SetLabel(2, "pump")

	return &StatusScreen2{
		softKeys: &softKeys,
		state:    state,
		config:   config,
	}
}

// Render updates the home screen, and provides an image
func (s *StatusScreen2) Render(img draw.Image) {
	Clear(img)
	DrawTxt(img, "Field: Field Name", 2, 8, tightpixel15.Font)
	DrawTxt(img, "Product: Product Name", 2, 22, tightpixel15.Font)
	DrawTxt(img, "99,999 gallons", 43, 37, tightpixel15.Font)

	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *StatusScreen2) Key(key isdata.Key) (ScreenID, interface{}) {
	switch key {
	case isdata.KeySK1:
		return ScreenIDHome, nil
	case isdata.KeyLeft:
		return ScreenIDStatus1, nil
	case isdata.KeyRight:
		return ScreenIDStatus3, nil
	}

	return ScreenIDNoChange, nil
}
