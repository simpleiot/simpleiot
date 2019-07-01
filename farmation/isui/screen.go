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
	ScreenIDPrev              // used by the "back" soft key
	ScreenIDHome
	ScreenIDStatus1
	ScreenIDStatus2
	ScreenIDStatus3
	ScreenIDMainMenu
	ScreenIDTankMenu1
	ScreenIDFieldMenu1
	ScreenIDOpMode1
	ScreenIDOpModeSetup
	ScreenIDTotals
	ScreenIDProductMenu1
	ScreenIDCalibration
	ScreenIDDiagConfig
	ScreenIDDiagInputs
	ScreenIDDiagOutputs
	ScreenIDDiagShutdownSettings
	ScreenIDDiagPulsesPres
	ScreenIDDiagLindsay
	ScreenIDDiagDevName
)

// Screens is a map of all screens in the system
type Screens struct {
	currentScreen ScreenID
	prevScreens   []ScreenID
	screens       map[ScreenID]Widget
	dialog        *DialogScreen
	state         *isdata.State
}

// Add a new screen
func (s *Screens) Add(ID ScreenID, screen Widget) {
	s.screens[ID] = screen
}

// NewScreens initializes all screens
func NewScreens(state *isdata.State, config *isdata.Config) *Screens {
	ret := &Screens{
		state: state,
	}

	ret.dialog = NewDialogScreen()

	ret.screens = make(map[ScreenID]Widget)
	ret.Add(ScreenIDHome, NewHomeScreen(state, config))
	ret.Add(ScreenIDStatus1, NewStatusScreen1(state, config))
	ret.Add(ScreenIDStatus2, NewStatusScreen2(state, config))
	ret.Add(ScreenIDStatus3, NewStatusScreen3(state, config))
	ret.Add(ScreenIDMainMenu, NewMainMenuScreen(state, config))
	ret.Add(ScreenIDTankMenu1, NewTankMenuScreen(state, config))
	ret.Add(ScreenIDFieldMenu1, NewFieldMenuScreen(state, config))
	ret.Add(ScreenIDOpMode1, NewOperatingModeScreen(state, config))
	ret.Add(ScreenIDOpModeSetup, NewOperatingModeSetupScreen(state, config))
	ret.Add(ScreenIDTotals, NewTotalsScreen(state, config))
	ret.Add(ScreenIDProductMenu1, NewProductMenuScreen(state, config))
	ret.Add(ScreenIDCalibration, NewCalibrationScreen(state, config))
	ret.Add(ScreenIDDiagConfig, NewDiagnosticsScreen(state, config))
	ret.Add(ScreenIDDiagInputs, NewDiagInputsScreen(state, config))
	ret.Add(ScreenIDDiagOutputs, NewDiagOutputsScreen(state, config))
	ret.Add(ScreenIDDiagShutdownSettings, NewDiagShutdownSettingsScreen(state, config))
	ret.Add(ScreenIDDiagPulsesPres, NewDiagPulsesPresScreen(state, config))
	ret.Add(ScreenIDDiagLindsay, NewDiagLindsayScreen(state, config))
	ret.Add(ScreenIDDiagDevName, NewDiagDevNameScreen(state, config))

	ret.currentScreen = ScreenIDHome

	return ret
}

// Render is used to draw a list of params, handles scrolling, etc.
func (s *Screens) Render(img draw.Image) {
	if s.state.DialogArm.Active {
		s.dialog.Render(img, s.state.DialogArm.Message)
	} else if s.state.DialogStateMachine.Active {
		s.dialog.Render(img, s.state.DialogStateMachine.Message)
	} else {
		s.screens[s.currentScreen].Render(img)
	}
}

// Key handles key input
func (s *Screens) Key(key isdata.Key) (ScreenID, interface{}, bool) {

	// dialogs
	if s.state.DialogArm.Active {
		if key == isdata.KeySK1 {
			return ScreenIDNoChange, isdata.UpdateDialogArmClose{}, true
		}

	} else if s.state.DialogStateMachine.Active {
		if key == isdata.KeySK1 {
			return ScreenIDNoChange, isdata.UpdateDialogAck{}, true
		}
	}

	// other screens
	screenID, action, handled := s.screens[s.currentScreen].Key(key)
	switch screenID {
	case ScreenIDNoChange:
	case ScreenIDPrev:
		s.currentScreen = s.prevScreens[len(s.prevScreens)-1] // go to prev screen
		s.prevScreens = s.prevScreens[:len(s.prevScreens)-1]  // remove screen from previous screens slice
	default:
		if len(s.prevScreens) < 200 {
			s.prevScreens = append(s.prevScreens, s.currentScreen)
		}
		s.currentScreen = screenID
	}

	return ScreenIDNoChange, action, handled
}
