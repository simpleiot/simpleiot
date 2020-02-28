package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
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
	ScreenIDModem
	ScreenIDDiagIPAddress
	ScreenIDDiagSIMImei
	ScreenIDDiag
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
	config        *isdata.Config
}

// Add a new screen
func (s *Screens) Add(ID ScreenID, screen Widget) {
	s.screens[ID] = screen
}

// NewScreens initializes all screens
func NewScreens(state *isdata.State, config *isdata.Config, db *isdb.IsDb) *Screens {
	ret := &Screens{
		state:  state,
		config: config,
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
	ret.Add(ScreenIDDiagSystemTime, NewDiagSystemTimeScreen(state, config))
	ret.Add(ScreenIDDiagSystemTimezone, NewDiagSystemTimezoneScreen(state, config))
	ret.Add(ScreenIDDiagPanel, NewDiagPanelScreen(state, config))
	ret.Add(ScreenIDModem, NewModemScreen(state, config))
	ret.Add(ScreenIDDiagIPAddress, NewDiagIPAddressScreen(state, config))
	ret.Add(ScreenIDDiagSIMImei, NewDiagSimImeiScreen(state, config))
	ret.Add(ScreenIDPanelType, NewPanelTypeScreen(state, config))

	ret.currentScreen = ScreenIDHome

	return ret
}

// Render is used to draw a list of params, handles scrolling, etc.
func (s *Screens) Render(img draw.Image) {

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
		renderHelpScreen(img, s.config.HelpScreen)
		return
	}

	s.screens[s.currentScreen].Render(img)
}

func renderHelpScreen(img draw.Image, helpScreen isdata.HelpScreen) {

	Clear(img)
	Heading(img, helpScreen.Heading+" - Help")
	font := tightpixel15.Font

	textLines := splitTextLines(helpScreen.Text)

	y := 13
	lineHeight := font.GetHeight()

	for _, line := range textLines {
		DrawTxt(img, line, 2, y, font)
		y += lineHeight + 1
	}

	// draw scroll bar if we have more than 1 screen
	if len(lines) > 4 {
		sbHeight := 50
		sbWidth := 4
		x := 123
		y := 8
		Rect(img, x, y, sbWidth, sbHeight)
		screenCount := (count + itemsPerScreen - 1) / itemsPerScreen
		blockHeight := sbHeight / screenCount

		// if divides scroll bar divides unevenly, fill up remaining space at the end
		if screen >= screenCount-1 {
			RectFilled(img, x, y+blockHeight*screen, sbWidth, blockHeight+sbHeight%screenCount)
		} else {
			RectFilled(img, x, y+blockHeight*screen, sbWidth, blockHeight)
		}
		// draw arrows
		if screen > 0 {
			Polyline(img,
				x, y,
				x+2, y-2,
				x+4, y)

			Polyline(img,
				x, y-1,
				x+2, y-3,
				x+4, y-1)
		}

		if screen < (screenCount - 1) {
			Polyline(img,
				x, y+sbHeight,
				x+2, y+sbHeight+2,
				x+4, y+sbHeight)

			Polyline(img,
				x, y+sbHeight+1,
				x+2, y+sbHeight+3,
				x+4, y+sbHeight+1)
		}
	}
}

func splitScreens(lines []string) (screens [][]string) {

	for i := 0; i < len(lines); i += 4 {
		screens = append(screens, lines[:i])
		lines = lines[i:]
	}

	return screens
}

func splitTextLines(s string) (lines []string) {

	font := tightpixel15.Font

	lineLen := 0

	line := ""

	for _, char := range s {

		line += string(char)

		_, charWidth := font.MeasureRune(char)
		lineLen += charWidth + 1

		if lineLen > 115 {
			iEnd := len(line) - 1
			charEnd := line[iEnd]
			line = line[:iEnd]
			lines = append(lines, line)
			line = string(charEnd)
			lineLen = 0
		}
	}

	return lines
}

// Key handles key input
func (s *Screens) Key(key isdata.Key) (ScreenID, interface{}, bool) {

	if key == isdata.KeySK1Release || key == isdata.KeySK1Hold {

		currentDialog, dialogKey := s.state.DialogHighestPriority()

		// If the dialog isn't nil (active dialogs), handle keys as coming
		// from the dialog
		if currentDialog != nil { // Take user directly to a screen that needs attention
			// when the dialog is closed
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
			}

			return ScreenIDNoChange, isdata.DialogClose{dialogKey}, true
		}

		if s.config.HelpScreen.Active {
			return ScreenIDNoChange, isdata.HelpScreenClose{}, true
		}
	}

	if key == isdata.KeyPump {
		s.switchScreen(ScreenIDPumpMode)
		return ScreenIDNoChange, nil, true
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
