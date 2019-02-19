package isui

import (
	"image/draw"
	"strconv"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// MenuItemType descripts the type of a field
type MenuItemType int

var menuSpacingText = 10
var menuSpacingValues = 11

// List of possible Param Types
const (
	MenuItemTypeScreen MenuItemType = iota
	MenuItemTypeInt
	MenuItemTypeFloat
	MenuItemTypeOnOff
	MenuItemTypeSelect
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
	showValues   bool
}

func (m *Menu) updateShowValues() {
	for _, item := range m.items {
		if item.Type != MenuItemTypeScreen && item.Type != MenuItemTypeSelect {
			m.showValues = true
			return
		}
	}

	m.showValues = false
}

// AddItemSelect adds a select list item
func (m *Menu) AddItemSelect(desc string) {
	m.items = append(m.items, MenuItem{
		Description: desc,
		Type:        MenuItemTypeSelect,
	})

	m.updateShowValues()
}

// AddItemScreen adds a screen selection to menu
func (m *Menu) AddItemScreen(desc string, s ScreenID) {
	m.items = append(m.items, MenuItem{
		Description: desc,
		Type:        MenuItemTypeScreen,
		Screen:      s,
	})

	m.updateShowValues()
}

// AddItemOnOff adds a on/off selection
func (m *Menu) AddItemOnOff(desc string, on bool) {
	m.items = append(m.items, MenuItem{
		Description: desc,
		Type:        MenuItemTypeOnOff,
		On:          on,
	})

	m.updateShowValues()
}

// AddItemInt adds an integer item to menu
func (m *Menu) AddItemInt(desc string, v float64) {
	m.items = append(m.items, MenuItem{
		Description: desc,
		Type:        MenuItemTypeInt,
		Value:       v,
	})

	m.updateShowValues()
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
	count := len(m.items)

	itemsPerScreen := 4
	screen := m.arrowPos / itemsPerScreen

	y := 13
	x := 2

	arrowScreenPos := m.arrowPos % itemsPerScreen

	start := itemsPerScreen * screen
	end := start + 3

	if end >= count {
		end = count - 1
	}

	if !m.showValues {
		x = 30
		Arrow(img, x-8, 17+arrowScreenPos*menuSpacingText)
	} else {
		Arrow(img, x+65, 17+arrowScreenPos*menuSpacingValues)
	}

	for i := start; i <= end; i++ {
		screenIndex := i - start
		item := m.items[i]
		offsetText := screenIndex * menuSpacingText
		offsetValues := screenIndex * menuSpacingValues
		DrawTxt(img, item.Description, x, y+offsetText, tightpixel15.Font)

		if m.showValues {
			// draw values
			Rect(img, 76, 12+offsetValues, 45, menuSpacingValues)
			switch item.Type {
			case MenuItemTypeScreen:
				DrawTxt(img, "open", 78, y+offsetValues, tightpixel15.Font)
			case MenuItemTypeSelect:
				DrawTxt(img, "select", 78, y+offsetValues, tightpixel15.Font)
			case MenuItemTypeInt:
				DrawTxtRight(img, strconv.Itoa(int(item.Value)), 120, y+1+offsetValues, tightpixel15.Font)
			case MenuItemTypeOnOff:
				DrawTxt(img, "on   off", 82, 13+offsetValues, tightpixel15.Font)
				top := 13 + offsetValues - 1
				bot := 13 + offsetValues + menuSpacingValues - 1
				switch item.On {
				case true:
					Line(img, 81, top, 81, bot)
					Line(img, 93, top, 93, bot)
				case false:
					Line(img, 102, top, 102, bot)
					Line(img, 118, top, 118, bot)
				}
			}
		}
	}
}

// MenuSelection is returned when a new item is selected
type MenuSelection string

// Key handles key input
func (m *Menu) Key(key isdata.Key) (ScreenID, interface{}) {
	switch key {
	case isdata.KeyUp:
		m.arrowPos--
		if m.arrowPos < 0 {
			m.arrowPos = 0
		}
	case isdata.KeyDown:
		m.arrowPos++
		if m.arrowPos >= len(m.items) {
			m.arrowPos = len(m.items) - 1
		}
	case isdata.KeyEnter:
		item := m.items[m.arrowPos]
		switch item.Type {
		case MenuItemTypeScreen:
			return item.Screen, nil
		case MenuItemTypeSelect:
			return ScreenIDNoChange, MenuSelection(item.Description)
		}
	}

	return ScreenIDNoChange, nil
}
