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
	ScreenIDDiagSystemTime
	ScreenIDDiagSystemTimezone
	ScreenIDDiagPanel
	ScreenIDDiagAdvancedOptions
	ScreenIDModem
	ScreenIDDiagIPAddress
	ScreenIDDiagSIMImei
	ScreenIDDiag
	ScreenIDPanelType
	ScreenIDGps
	ScreenIDSaver
	ScreenIDStorage
)

// Screens is a map of all screens in the system
type Screens struct {
	currentScreen ScreenID
	prevScreens   []ScreenID
	screens       map[ScreenID]Widget
	dialog        *DialogScreen
	dialogArmReq  *DialogArmReqScreen
	helpScreen    *HelpScreenUI
	state         *isdata.State
	config        *isdata.Config
	screenSaver   bool
}

// Add a new screen
func (s *Screens) Add(ID ScreenID, screen Widget) {
	s.screens[ID] = screen
}

// ScreenSaver control screen saver
func (s *Screens) ScreenSaver(enable bool) {
	if enable != s.screenSaver {
		if enable {
			s.switchScreen(ScreenIDSaver)
		} else {
			s.switchScreen(ScreenIDHome)
		}
	}
	s.screenSaver = enable
}

// NewScreens initializes all screens
func NewScreens(state *isdata.State, config *isdata.Config, db *isdb.IsDb) *Screens {
	ret := &Screens{
		state:        state,
		config:       config,
		dialog:       NewDialogScreen(),
		dialogArmReq: NewDialogArmReqScreen(config, state),
		helpScreen:   NewHelpScreenUI("", [][]string{{""}}),
	}

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
	ret.Add(ScreenIDDiagSystemTime, NewDiagSystemTimeScreen(state, config))
	ret.Add(ScreenIDDiagSystemTimezone, NewDiagSystemTimezoneScreen(state, config))
	ret.Add(ScreenIDDiagPanel, NewDiagPanelScreen(state, config))
	ret.Add(ScreenIDDiagAdvancedOptions, NewDiagAdvancedOptionsScreen(state, config))
	ret.Add(ScreenIDModem, NewModemScreen(state, config))
	ret.Add(ScreenIDDiagIPAddress, NewDiagIPAddressScreen(state, config))
	ret.Add(ScreenIDDiagSIMImei, NewDiagSimImeiScreen(state, config))
	ret.Add(ScreenIDPanelType, NewPanelTypeScreen(state, config))
	ret.Add(ScreenIDGps, NewGpsScreen(state, config))
	ret.Add(ScreenIDSaver, NewScreenSaver())
	ret.Add(ScreenIDStorage, NewStorageScreen(state, config, db))

	ret.currentScreen = ScreenIDHome

	return ret
}

// Render is used to draw a list of params, handles scrolling, etc.
func (s *Screens) Render(img draw.Image) {
	if !s.screenSaver {
		currentDialog, _ := s.state.DialogHighestPriority()

		// If the dialog isn't nil (there are active dialogs), render the
		// returned active dialog
		if currentDialog != nil {
			if currentDialog.ID == isdata.DialogArmReq {
				// ArmRequirements dialog requires a special render method
				s.dialogArmReq.Render(img)
				return
			}

			s.dialog.Render(img, currentDialog)
			return
		}

		// If the user has activated the help screen
		if s.config.HelpScreen.Active {
			s.helpScreen.UpdateContent(s.config.HelpScreen)
			s.helpScreen.Render(img)
			return
		}
	}

	s.screens[s.currentScreen].Render(img)
}

// Key handles key input
func (s *Screens) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	if !s.screenSaver {
		currentDialog, dialogKey := s.state.DialogHighestPriority()

		// Prioritize this key because we want to use it from any part
		// of the system, including dialogs and help screens
		if key == isdata.KeyPump || key == isdata.KeyPumpHold {

			switch key {
			case isdata.KeyPump: // Pump Screen
				if s.currentScreen != ScreenIDPumpMode {
					s.switchScreen(ScreenIDPumpMode)
				}
			case isdata.KeyPumpHold: // Diagnostics Flow/Pressure
				if s.currentScreen != ScreenIDDiagPulsesPres {
					s.switchScreen(ScreenIDDiagPulsesPres)
				}

			}

			if currentDialog != nil {
				return ScreenIDNoChange, isdata.DialogCancel{dialogKey}, true
			}
			if s.config.HelpScreen.Active {
				return ScreenIDNoChange, isdata.HelpScreenClose{}, true
			}
			return ScreenIDNoChange, nil, true
		}

		// If the dialog isn't nil (active dialogs), handle keys as coming
		// from the dialog
		if currentDialog != nil {

			if currentDialog.ID == isdata.DialogArmReq {
				_, _, handled := s.dialogArmReq.Key(key)
				if handled {
					return ScreenIDNoChange, nil, true
				}
			}

			switch key {
			case isdata.KeySK1Release: // OK

				// Take user directly to a screen that needs attention
				switch currentDialog.ID {
				case isdata.DialogArm:
					switch currentDialog.Message {
					case "Cannot arm in Monitor \nOnly mode, please switch \nmodes":
						s.switchScreen(ScreenIDOpMode1)
					case "Pump Command \nInput not selected, please \nselect before arming":
						s.switchScreen(ScreenIDPumpMode)
					}
				case isdata.DialogArmReq:
					if s.state.InputInjector == isdata.InputStateOff &&
						s.config.UserPumpMode == isdata.UserPumpModeOff {
						s.switchScreen(ScreenIDPumpMode)
					}
				case isdata.DialogHistoryData:
					s.switchScreen(ScreenIDStorage)
				}

				// Close the dialog
				return ScreenIDNoChange, isdata.DialogClose{dialogKey}, true

			case isdata.KeySK2: // Cancel
				if currentDialog.CancelActivated {
					return ScreenIDNoChange, isdata.DialogCancel{dialogKey}, true
				}
			}

			return ScreenIDNoChange, nil, true
		}

		if s.config.HelpScreen.Active {

			return ScreenIDNoChange, s.helpScreen.Key(key), true

		}
	}

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
