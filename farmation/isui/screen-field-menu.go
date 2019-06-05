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
		textEntryScreen: NewTextEntryScreen(true, true),
	}
}

// Render updates the home screen, and provides an image
func (s *FieldMenuScreen) Render(img draw.Image) {
	Clear(img)

	s.menu.ResetItems()
	for i, fieldConfig := range s.config.FieldConfigs {
		selected := (i == s.config.CurrentFieldIndex)
		s.menu.AddItemSelect(fieldConfig.Description, isdata.UpdateCurrentFieldIndex(s.menu.GetArrowPos()), selected)
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
	if s.edit { // passes key inputs to textEntryScreen and follows returned commands
		command := s.textEntryScreen.Key(key)
		switch command {
		case TextEntryCommandNone: // do nothing
		case TextEntryCommandSave: //save
			s.exitEdit()
			return ScreenIDNoChange, isdata.UpdateFieldName{
				Index: s.menu.GetArrowPos(),
				Name:  s.textEntryScreen.GetTextEdit(),
			}, true

		case TextEntryCommandCancel: // cancel
			s.exitEdit()
		}
	} else {
		switch key {
		case isdata.KeySK1: // Back
			s.menu.ResetArrowPos() // return arrow to top of screen
			return ScreenIDMainMenu, nil, true
		case isdata.KeySK2: // Edit
			s.edit = true
			s.textEntryScreen.txtEdit = s.menu.Description()
			s.textEntryScreen.inputChars.IndexTo(s.textEntryScreen.txtEdit[s.textEntryScreen.cursorPos]) // move inputChars cursor to current pos in txtEdit
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
