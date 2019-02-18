package isui

import (
	"image/draw"
	"strconv"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// MenuItemType descripts the type of a field
type MenuItemType int

// List of possible Param Types
const (
	MenuItemTypeScreen MenuItemType = iota
	MenuItemTypeInt
	MenuItemTypeFloat
	MenuItemTypeOnOff
)

// MenuItem describes a field that is displayed
type MenuItem struct {
	Description string
	Type        MenuItemType
	Screen      ScreenID
	Value       float64
	On          bool
	Precision   int
}

// Menu descripes a list user selectable options
type Menu struct {
	items        []MenuItem
	scrollOffset int
	arrowPos     int
}

// AddItemScreen adds a screen selection to menu
func (m *Menu) AddItemScreen(desc string, s ScreenID) {
	m.items = append(m.items, MenuItem{
		Description: desc,
		Type:        MenuItemTypeScreen,
		Screen:      s,
	})
}

// AddItemOnOff adds a on/off selection
func (m *Menu) AddItemOnOff(desc string, on bool) {
	m.items = append(m.items, MenuItem{
		Description: desc,
		Type:        MenuItemTypeOnOff,
		On:          on,
	})
}

// AddItemInt adds an integer item to menu
func (m *Menu) AddItemInt(desc string, v float64) {
	m.items = append(m.items, MenuItem{
		Description: desc,
		Type:        MenuItemTypeInt,
		Value:       v,
	})
}

// SetValue is used to set a parameter value
func (m *Menu) SetValue(desc string, v float64) {
	for i, item := range m.items {
		if desc == item.Description {
			m.items[i].Value = v
			break
		}
	}
}

// Render is used to draw a list of params, handles scrolling, etc.
func (m *Menu) Render(img draw.Image) {
	for i, item := range m.items {
		offset := i * menuSpacingTight
		DrawTxt(img, item.Description, 2, 13+offset, tightpixel15.Font)
		Rect(img, 76, 12+offset, 45, menuSpacingTight)
		switch item.Type {
		case MenuItemTypeInt:
			DrawTxt(img, strconv.Itoa(int(item.Value)), 78, 13+offset, tightpixel15.Font)
		case MenuItemTypeOnOff:
			DrawTxt(img, "on   off", 82, 13+offset, tightpixel15.Font)
			top := 13 + offset - 1
			bot := 13 + offset + menuSpacingTight - 1
			switch item.On {
			case true:
				Line(img, 81, top, 81, bot)
				Line(img, 93, top, 93, bot)
			case false:
				Line(img, 102, top, 102, bot)
				Line(img, 118, top, 118, bot)
			}
		}
		Arrow(img, 67, 17+m.arrowPos*menuSpacingTight)
	}
}

// Key handles key input
func (m *Menu) Key(key isdata.Key) (ScreenID, interface{}) {
	switch key {
	case isdata.KeyUp:
		m.arrowPos--
		if m.arrowPos < 0 {
			m.arrowPos = len(m.items) - 1
		}
	case isdata.KeyDown:
		m.arrowPos++
		if m.arrowPos >= len(m.items) {
			m.arrowPos = 0
		}
	case isdata.KeyEnter:
		item := m.items[m.arrowPos]
		switch item.Type {
		case MenuItemTypeScreen:
			return item.Screen, nil
		}
	}

	return ScreenNoChange, nil
}
