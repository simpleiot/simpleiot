package isui

import (
	"image/draw"
	"strconv"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// StorageScreen displays storage stats and offers data purge option
type StorageScreen struct {
	softKeys *SoftKeys
	state    *isdata.State
	config   *isdata.Config
	menu     Menu
}

// NewStorageScreen creates a modem information screen
func NewStorageScreen(state *isdata.State, config *isdata.Config) *StorageScreen {
	isdata.InitState(state) // make sure that ProductStates and FieldStates arrays are large enough
	menu := Menu{}

	return &StorageScreen{
		// update this from sample screen
		softKeys: NewSoftKeys("back", "purge"),
		state:    state,
		config:   config,
		menu:     menu,
	}
}

// Render updates the screen, and provides an image
func (s *StorageScreen) Render(img draw.Image) {
	Clear(img)
	s.menu.ResetItems()

	Heading(img, "Data Storage Admin")

	//s.menu.AddItemOnOff("Enable Modem", !s.config.ModemDisabled, isdata.UpdateModemDisabled(!s.config.ModemDisabled))
	//s.menu.AddItemString("Connected", connected)
	s.menu.AddItemStringRight("Record Count", strconv.Itoa(s.state.DbSampleCount))
	s.menu.AddItemStringRight("Data Usage", strconv.FormatFloat(s.state.DataUsage, 'f', 0, 64)+"%")
	s.menu.AddItemStringRight("Rootfs Usage", strconv.FormatFloat(s.state.RootUsage, 'f', 0, 64)+"%")
	s.menu.Render(img)
	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *StorageScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1Hold: // Back key held -> Home screen
		s.menu.ResetArrowPos() // return arrow to top of screen
		return ScreenIDHome, nil, true
	case isdata.KeySK1Release: // Back
		s.menu.ResetArrowPos() // return arrow to top of screen
		return ScreenIDPrev, nil, true
	case isdata.KeySK2:
		// FIXME implement purge
	case isdata.KeyUp, isdata.KeyUpHold, isdata.KeyDown, isdata.KeyDownHold, isdata.KeyRight, isdata.KeyRightHold, isdata.KeyLeft, isdata.KeyLeftHold, isdata.KeyEnter, isdata.KeyEnterHold:
		return s.menu.Key(key)
	}

	return ScreenIDNoChange, nil, true
}
