package isui

import (
	"fmt"
	"image/draw"
	"strings"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// HelpScreenUI stores the current content of the help screen
// as well as the part of the text that should be displayed
// if it won't all fit on the lcd. If the content changes
// in config.HelpScreen (we can tell this by comparing the
// Name field in the config with the Name field in this
// struct) the content stored here is updated
// and the text is split into lines and screens stored in
// Screens
type HelpScreenUI struct {
	Name    string
	Screens [][]string

	// This is an index that tells the Render
	// method what part of the help screen text
	// to draw if there is more than will fit
	// on the lcd. Also used for scroll bar.
	Index    int
	softKeys *SoftKeys
}

// NewHelpScreenUI initializes this structure. It must be used so that
// the value of this pointer is not nil.
func NewHelpScreenUI() *HelpScreenUI {
	return &HelpScreenUI{
		softKeys: NewSoftKeys("back"),
	}
}

// UpdateContent compares the Name field in the configHelpScreen parameter with
// the Name field in HelpScreenUI. If they are not the same, it splits the Text
// field from configHelpScreen into lines and screens and stores it in the Screens
// field in HelpScreenUI. Then it resets Index to zero, and updates the Name.
func (h *HelpScreenUI) UpdateContent(configHelpScreen isdata.HelpScreen) {

	if h.Name == configHelpScreen.Name {
		return
	}

	h.Screens = splitScreens(splitTextLines(configHelpScreen.Text))
	h.Index = 0
	h.Name = configHelpScreen.Name
}

// Render renders the portion of the screen designated by Index
func (h *HelpScreenUI) Render(img draw.Image) {

	Clear(img)

	Heading(img, h.Name+" - Help")

	h.softKeys.Render(img, 0, 54)

	font := tightpixel15.Font

	numScreens := len(h.Screens)

	if numScreens <= 0 {
		return
	}

	y := 12
	lineHeight := font.GetHeight()

	for _, line := range h.Screens[h.Index] {
		DrawTxt(img, line, 2, y, font)
		y += lineHeight + 1
	}

	// Draw scroll bar
	if numScreens > 1 {
		sbHeight := 50
		sbWidth := 4
		x := 123
		y := 8
		Rect(img, x, y, sbWidth, sbHeight)
		screenCount := len(h.Screens)
		blockHeight := sbHeight / screenCount

		// if divides scroll bar divides unevenly, fill up remaining space at the end
		if h.Index >= screenCount-1 {
			RectFilled(img, x, y+blockHeight*h.Index, sbWidth, blockHeight+sbHeight%screenCount)
		} else {
			RectFilled(img, x, y+blockHeight*h.Index, sbWidth, blockHeight)
		}
		// draw arrows
		if h.Index > 0 {
			Polyline(img,
				x, y,
				x+2, y-2,
				x+4, y)

			Polyline(img,
				x, y-1,
				x+2, y-3,
				x+4, y-1)
		}

		if h.Index < (screenCount - 1) {
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

	for i := 4; i < len(lines); i = i {
		screens = append(screens, lines[:i])
		//fmt.Println("COLLIN2:", screens)
		lines = lines[i:]
	}
	//fmt.Println("COLLIN3:", append(screens, lines))

	return append(screens, lines)
}

func splitTextLines(s string) (lines []string) {

	words := strings.SplitN(s, " ", 1000)

	linePixels := 0

	var line []string

	for _, w := range words {
		wordPixels := tightpixel15.Font.MeasureString(w + " ")
		linePixels += wordPixels

		if linePixels > 122 {
			lines = append(lines, strings.Join(line, " "))
			line = nil
			linePixels = wordPixels
		}

		line = append(line, w)
		fmt.Println(line, linePixels)
	}

	lines = append(lines, strings.Join(line, " "))

	/*
		for _, line := range lines {
			fmt.Println("COLLIN, line:", line)
		}
	*/

	return lines
}
