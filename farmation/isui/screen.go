package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// ScreenID is a constant that identifies a screen
type ScreenID int

// Define constants for various screens
const (
	ScreenIDNoChange ScreenID = iota
	ScreenIDHome
	ScreenIDStatus1
	ScreenIDStatus2
	ScreenIDStatus3
	ScreenIDMainMenu
	ScreenIDTankMenu1
	ScreenIDFieldMenu1
	ScreenIDEditFieldNames
	ScreenIDOpMode1
	ScreenIDOpModeSetup
	ScreenIDTotals
	ScreenIDProductMenu1
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

	ret[ScreenIDHome] = NewHomeScreen(state, config)
	ret[ScreenIDStatus1] = NewStatusScreen1(state, config)
	ret[ScreenIDStatus2] = NewStatusScreen2(state, config)
	ret[ScreenIDStatus3] = NewStatusScreen3(state, config)
	ret[ScreenIDMainMenu] = NewMainMenuScreen(state, config)
	ret[ScreenIDTankMenu1] = NewTankMenuScreen(state, config)
	ret[ScreenIDFieldMenu1] = NewFieldMenuScreen(state, config)
	ret[ScreenIDOpMode1] = NewOperatingModeScreen(state, config)
	ret[ScreenIDOpModeSetup] = NewOperatingModeSetupScreen(state, config)
	ret[ScreenIDTotals] = NewTotalsScreen(state, config)
	ret[ScreenIDProductMenu1] = NewProductMenuScreen(state, config)

	return

}
