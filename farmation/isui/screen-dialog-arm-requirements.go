package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// DialogArmReqScreen is used to display modal dialog messages
type DialogArmReqScreen struct {
	config            *isdata.Config
	state             *isdata.State
	softKeys          *SoftKeys
	helpScreenActive  bool
	helpScreen        *HelpScreenUI
	helpScreenContent [][]string
}

// NewDialogArmReqScreen creates a new dialog screen
func NewDialogArmReqScreen(config *isdata.Config, state *isdata.State) *DialogArmReqScreen {
	return &DialogArmReqScreen{
		config:     config,
		state:      state,
		softKeys:   NewSoftKeys("OK"),
		helpScreen: NewHelpScreenUI(),
		helpScreenContent: SplitScreens(SplitTextLines("Irrigator, Water and Injector " +
			"Command need to be on in order to arm. If Injector Command is not on be sure " +
			"that correct pump command source is selected. This is accessed by the pump " +
			"key. Flow rate needs to be greater than 5 and base pressure needs to be higher " +
			"than start pressure to arm. Start pressure can be changed or disabled in the " +
			"Operating Mode Setup screen.")),
	}
}

// Render screen
func (s *DialogArmReqScreen) Render(img draw.Image) {

	if !s.helpScreenActive {

		Clear(img)

		Heading(img, "Required to Arm")

		font := tightpixel15.Font
		y1, y2, y3 := 13, 26, 40
		x1, x2, x3, x4 := 2, 47, 64, 112

		// Requirement descriptions
		DrawTxt(img, "Injector", x1, y1, font)
		DrawTxt(img, "Water On", x1, y2, font)
		DrawTxt(img, "Irrigator", x1, y3, font)
		DrawTxt(img, "Flow", x3, y1, font)
		DrawTxt(img, "Pressure", x3, y2, font)

		// Requirements met: check mark or X
		drawMet(s.config, s.state, img, 0, x2, y1)
		drawMet(s.config, s.state, img, 3, x4, y1)
		drawMet(s.config, s.state, img, 1, x2, y2)
		drawMet(s.config, s.state, img, 4, x4, y2)
		drawMet(s.config, s.state, img, 2, x2, y3)

		_, _ = x2, x4

		s.softKeys.Render(img, 0, 54)

	} else {
		s.helpScreen.Render(img)
	}
}

// drawMet draws a check mark if ArmReqMet()[i] and an X
// if !ArmReqMet()[i]
func drawMet(config *isdata.Config, state *isdata.State, img draw.Image, i, x, y int) {
	met := isdata.ArmReqMet(config, state)[i]
	if met {
		Polyline(img,
			x, y+4,
			x+3, y+7,
			x+9, y-2)
	} else {
		Line(img, x, y, x+8, y+8)
		Line(img, x+8, y, x, y+8)
	}
}
