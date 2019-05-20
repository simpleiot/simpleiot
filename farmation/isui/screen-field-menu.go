package isui

import (
	"image/draw"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// FieldMenuScreen is used to display status info
type FieldMenuScreen struct {
	textEntryScreen *TextEntryScreen
	softKeys        *SoftKeys
	state           *isdata.State
	config          *isdata.Config
	menu            *Menu
	edit            bool
}

// NewFieldMenuScreen initializes and returns a HomeScreen
func NewFieldMenuScreen(state *isdata.State, config *isdata.Config) *FieldMenuScreen {
	return &FieldMenuScreen{
		softKeys:        NewSoftKeys("back", "edit", "import"),
		state:           state,
		config:          config,
		menu:            NewMenu(),
		textEntryScreen: NewTextEntryScreen(),
	}
}

// Render updates the home screen, and provides an image
func (s *FieldMenuScreen) Render(img draw.Image) {
	Clear(img)

	s.menu.ResetItems()
	for _, fieldConfig := range s.config.FieldConfigs {
		s.menu.AddItemScreen(fieldConfig.Description, ScreenIDNoChange)
	}

	if s.edit { // render text entry screen
		s.textEntryScreen.Render(img)
	} else { // render regular screen
		Heading(img, "Field Menu")
		s.menu.Render(img)
		s.softKeys.Render(img, 0, 54)
	}
}

// Key processes key inputs to this screen
func (s *FieldMenuScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	if s.edit { //handles some key inputs for txt entry screen and passes rest to textEntryScreen
		switch key {
		case isdata.KeySK1: //save
			s.exitEdit()
			return ScreenIDNoChange, isdata.UpdateFieldName{
				Index: s.menu.GetArrowPos(),
				Name:  s.textEntryScreen.GetTextEdit(),
			}, true

		case isdata.KeySK4: // cancel
			s.exitEdit()
		case isdata.KeySK2, isdata.KeySK3, isdata.KeyEnter, isdata.KeyRight, isdata.KeyLeft, isdata.KeyUp, isdata.KeyDown:
			s.textEntryScreen.Key(key)
		}
	} else {
		switch key {
		case isdata.KeySK1:
			s.menu.ResetArrowPos() // return arrow to top of screen
			return ScreenIDMainMenu, nil, true
		case isdata.KeySK2:
			s.edit = true
			s.textEntryScreen.txtEdit = s.menu.Description()
		case isdata.KeyUp, isdata.KeyDown, isdata.KeyRight, isdata.KeyLeft, isdata.KeyEnter:
			return s.menu.Key(key)
		}
	}

	return ScreenIDNoChange, nil, true
}

func (s *FieldMenuScreen) exitEdit() {
	s.edit = false
	s.textEntryScreen.ExitEdit()
}
