package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// ScreenID is a constant that identifies a screen
type ScreenID int

// Define constants for various screens
const (
	ScreenNoChange = iota
	ScreenHome
	ScreenStatus1
	ScreenStatus2
	ScreenStatus3
	ScreenMainMenu
	ScreenTankMenu1
	ScreenFieldMenu1
	ScreenEditFieldNames
	ScreenOpMode1
	ScreenOpModeSetup
)

// Screen is an interface that generally represents a screen
type Screen interface {
	Render(img draw.Image)
	Key(isdata.Key) (ScreenID, interface{})
}

// Screens is a map of all screens in the system
type Screens map[ScreenID]Screen

// InitScreens initializes all screens
func InitScreens(state *isdata.State, config *isdata.Config) (ret Screens) {
	ret = make(map[ScreenID]Screen)

	ret[ScreenHome] = NewHomeScreen(state, config)
	ret[ScreenStatus1] = NewStatusScreen1(state, config)
	ret[ScreenStatus2] = NewStatusScreen2(state, config)
	ret[ScreenStatus3] = NewStatusScreen3(state, config)

	return

}
