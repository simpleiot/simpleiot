package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/farmation/isdb"
)

// ScreenID is a constant that identifies a screen
type ScreenID int

// Define constants for various screens
const (
	ScreenIDNoChange ScreenID = iota
	ScreenIDPrev              // used by the "back" soft key
	ScreenIDHome
	ScreenIDFaultsActive
	ScreenIDFaultsHistory
	ScreenIDStatus1
	ScreenIDStatus2
	ScreenIDStatus3
	ScreenIDMainMenu
	ScreenIDTankMenu1
	ScreenIDFieldMenu1
	ScreenIDOpMode1
	ScreenIDOpModeSetup
	ScreenIDPumpMode
	ScreenIDPumpTest
	ScreenIDTotals
	ScreenIDProductMenu1
	ScreenIDCalibration
	ScreenIDDiagConfig
	ScreenIDDiagInputs
	ScreenIDDiagOutputs
	ScreenIDDiagPulsesPres
	ScreenIDPulseOutputTest
	ScreenIDDiagLindsay
	ScreenIDDiagDevName
	ScreenIDDiagPanel
	ScreenIDModem
	ScreenIDDiagIPAddress
	ScreenIDPanelType
)

// Screens is a map of all screens in the system
type Screens struct {
	currentScreen ScreenID
	prevScreens   []ScreenID
	screens       map[ScreenID]Widget
	dialog        *DialogScreen
	dialogArmReq  *DialogArmReqScreen
	state         *isdata.State
}

// Add a new screen
func (s *Screens) Add(ID ScreenID, screen Widget) {
	s.screens[ID] = screen
}

// NewScreens initializes all screens
func NewScreens(state *isdata.State, config *isdata.Config, db *isdb.IsDb) *Screens {
	ret := &Screens{
		state: state,
	}

	ret.dialog = NewDialogScreen()
	ret.dialogArmReq = NewDialogArmReqScreen(config, state)

	ret.screens = make(map[ScreenID]Widget)
	ret.Add(ScreenIDHome, NewHomeScreen(state, config))
	ret.Add(ScreenIDFaultsActive, NewFaultsActiveScreen(state, config))
	ret.Add(ScreenIDFaultsHistory, NewFaultsHistoryScreen(state, config, db))
	ret.Add(ScreenIDStatus1, NewStatusScreen1(state, config))
	ret.Add(ScreenIDStatus2, NewStatusScreen2(state, config))
	ret.Add(ScreenIDStatus3, NewStatusScreen3(state, config))
	ret.Add(ScreenIDMainMenu, NewMainMenuScreen(state, config))
	ret.Add(ScreenIDTankMenu1, NewTankMenuScreen(state, config))
	ret.Add(ScreenIDFieldMenu1, NewFieldMenuScreen(state, config))
	ret.Add(ScreenIDOpMode1, NewOperatingModeScreen(state, config))
	ret.Add(ScreenIDOpModeSetup, NewOperatingModeSetupScreen(state, config))
	ret.Add(ScreenIDPumpMode, NewPumpModeScreen(state, config))
	ret.Add(ScreenIDPumpTest, NewPumpTestScreen(state, config))
	ret.Add(ScreenIDTotals, NewTotalsScreen(state, config))
	ret.Add(ScreenIDProductMenu1, NewProductMenuScreen(state, config))
	ret.Add(ScreenIDCalibration, NewCalibrationScreen(state, config))
	ret.Add(ScreenIDDiagConfig, NewDiagnosticsScreen(state, config))
	ret.Add(ScreenIDDiagInputs, NewDiagInputsScreen(state, config))
	ret.Add(ScreenIDDiagOutputs, NewDiagOutputsScreen(state, config))
	ret.Add(ScreenIDDiagPulsesPres, NewDiagPulsesPresScreen(state, config))
	ret.Add(ScreenIDPulseOutputTest, NewPulseOutputTestScreen(state, config))
	ret.Add(ScreenIDDiagLindsay, NewDiagLindsayScreen(state, config))
	ret.Add(ScreenIDDiagDevName, NewDiagDevNameScreen(state, config))
	ret.Add(ScreenIDDiagPanel, NewDiagPanelScreen(state, config))
	ret.Add(ScreenIDModem, NewModemScreen(state, config))
	ret.Add(ScreenIDDiagIPAddress, NewDiagIPAddressScreen(state, config))
	ret.Add(ScreenIDPanelType, NewPanelTypeScreen(state, config))

	ret.currentScreen = ScreenIDHome

	return ret
}

// Render is used to draw a list of params, handles scrolling, etc.
func (s *Screens) Render(img draw.Image) {
	switch {
	case s.state.DialogReboot.Active:
		s.dialog.Render(img, s.state.DialogReboot.Message)
	case s.state.DialogUpdate.Active:
		s.dialog.Render(img, s.state.DialogUpdate.Message)
	case s.state.DialogInvalidPanel.Active:
		s.dialog.Render(img, s.state.DialogInvalidPanel.Message)
	case s.state.DialogUnknownVisionState.Active:
		s.dialog.Render(img, s.state.DialogUnknownVisionState.Message)
	case s.state.DialogExport.Active:
		s.dialog.Render(img, s.state.DialogExport.Message)
	case s.state.DialogArm.Active:
		s.dialog.Render(img, s.state.DialogArm.Message)
	case s.state.DialogArmInputs.Active:
		s.dialog.Render(img, s.state.DialogArmInputs.Message)
	case s.state.DialogArmReq.Active:
		s.dialogArmReq.Render(img, s.state.DialogArmReq.Message)
	case s.state.DialogStateMachine.Active:
		s.dialog.Render(img, s.state.DialogStateMachine.Message)
	case s.state.DialogApp.Active:
		s.dialog.Render(img, s.state.DialogApp.Message)
	default:
		s.screens[s.currentScreen].Render(img)
	}
}

// Key handles key input
func (s *Screens) Key(key isdata.Key) (ScreenID, interface{}, bool) {

	// dialogs
	// Note, below needs to be the same order as in render
	switch {
	case s.state.DialogUpdate.Active:
		if key == isdata.KeySK1Release || key == isdata.KeySK1Hold {
			return ScreenIDNoChange, isdata.UpdateDialogUpdateClose{}, true
		}
		return ScreenIDNoChange, nil, true
	case s.state.DialogInvalidPanel.Active:
		if key == isdata.KeySK1Release || key == isdata.KeySK1Hold {
			return ScreenIDNoChange, isdata.UpdateDialogInvalidPanelClose{}, true
		}
		return ScreenIDNoChange, nil, true
	case s.state.DialogUnknownVisionState.Active:
		if key == isdata.KeySK1Release || key == isdata.KeySK1Hold {
			return ScreenIDNoChange, isdata.UpdateDialogUnknownVisionStateClose{}, true
		}
		return ScreenIDNoChange, nil, true
	case s.state.DialogExport.Active:
		if key == isdata.KeySK1Release || key == isdata.KeySK1Hold {
			return ScreenIDNoChange, isdata.UpdateDialogExportClose{}, true
		}
		return ScreenIDNoChange, nil, true
	case s.state.DialogArm.Active:
		if key == isdata.KeySK1Release || key == isdata.KeySK1Hold {
			s.switchScreen(ScreenIDOpMode1)
			return ScreenIDNoChange, isdata.UpdateDialogArmClose{}, true
		}
		return ScreenIDNoChange, nil, true
	case s.state.DialogArmInputs.Active:
		if key == isdata.KeySK1Release || key == isdata.KeySK1Hold {
			s.switchScreen(ScreenIDPumpMode)
			return ScreenIDNoChange, isdata.UpdateDialogArmInputsClose{}, true
		}
		return ScreenIDNoChange, nil, true
	case s.state.DialogArmReq.Active:
		if key == isdata.KeySK1Release || key == isdata.KeySK1Hold {
			return ScreenIDNoChange, isdata.UpdateDialogArmReqClose{}, true
		}
		return ScreenIDNoChange, nil, true
	case s.state.DialogStateMachine.Active:
		if key == isdata.KeySK1Release || key == isdata.KeySK1Hold {
			return ScreenIDNoChange, isdata.UpdateDialogStateMachineClose{}, true
		}
		return ScreenIDNoChange, nil, true
	case s.state.DialogApp.Active:
		if key == isdata.KeySK1Release || key == isdata.KeySK1Hold {
			return ScreenIDNoChange, isdata.UpdateDialogAppClose{}, true
		}
		return ScreenIDNoChange, nil, true
	}

	// other screens
	screenID, action, handled := s.screens[s.currentScreen].Key(key)
	switch screenID {
	case ScreenIDNoChange:
	case ScreenIDPrev:
		s.currentScreen = s.prevScreens[len(s.prevScreens)-1] // go to prev screen
		s.prevScreens = s.prevScreens[:len(s.prevScreens)-1]  // remove screen from previous screens slice
	default:
		s.switchScreen(screenID)
	}

	// if at home screen, empty prevScreens array
	if s.currentScreen == ScreenIDHome {
		s.prevScreens = nil
	}

	return ScreenIDNoChange, action, handled
}

func (s *Screens) switchScreen(id ScreenID) {
	// add current screen to prevScreens array
	if len(s.prevScreens) < 200 {
		s.prevScreens = append(s.prevScreens, s.currentScreen)
	}

	// move to new screen
	s.currentScreen = id
}
