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
	MenuItemString
	MenuItemTypeInt
	MenuItemTypeFloat
	MenuItemTypeOnOff
	MenuItemTypeSelect
	MenuItemTypeCommand
)

// MenuItem describes a field that is displayed
type MenuItem struct {
	Description string
	Type        MenuItemType
	Screen      ScreenID
	ValueInt    int
	Value       float64
	ValueString string
	On          bool
	Precision   int
	Message     interface{}
}

// Menu descripes a list user selectable options
type Menu struct {
	items        []MenuItem
	scrollOffset int
	arrowPos     int
	showValues   bool
}

// NewMenu creates a new menu
func NewMenu() *Menu {
	return &Menu{}
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

// ResetItems clears the menu items
func (m *Menu) ResetItems() {
	m.items = []MenuItem{}
}

// Description returns the menu item description
func (m *Menu) Description() string {
	return m.items[m.arrowPos].Description
}

// ValueInt returns the menu item value (integer)
func (m *Menu) ValueInt() int {
	return m.items[m.arrowPos].ValueInt
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

// BoolToString turns a boolean value into "on"/"off"
func BoolToString(val bool) string {
	if val {
		return "on"
	}
	return "off"
}

// AddItemOnOff adds a on/off selection
func (m *Menu) AddItemOnOff(desc string, on bool, msg interface{}) {
	m.items = append(m.items, MenuItem{
		Description: desc,
		Type:        MenuItemTypeOnOff,
		On:          on,
		Message:     msg,
	})

	m.updateShowValues()
}

// AddItemString adds a select list item
func (m *Menu) AddItemString(desc string, value string) {
	m.items = append(m.items, MenuItem{
		Description: desc,
		ValueString: value,
		Type:        MenuItemString,
	})

	m.updateShowValues()
}

// AddItemInt adds an integer item to menu
func (m *Menu) AddItemInt(desc string, v int) {
	m.items = append(m.items, MenuItem{
		Description: desc,
		Type:        MenuItemTypeInt,
		ValueInt:    v,
	})

	m.updateShowValues()
}

// AddItemFloat adds an integer item to menu
func (m *Menu) AddItemFloat(desc string, v float64) {
	m.items = append(m.items, MenuItem{
		Description: desc,
		Type:        MenuItemTypeFloat,
		Value:       v,
	})

	m.updateShowValues()
}

// AddItemCommand adds a command menu item
func (m *Menu) AddItemCommand(desc string, msg interface{}) {
	m.items = append(m.items, MenuItem{
		Description: desc,
		Type:        MenuItemTypeCommand,
		Message:     msg,
	})

	m.updateShowValues()
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
			case MenuItemTypeCommand:
				DrawTxt(img, "start", 78, y+offsetValues, tightpixel15.Font)
			case MenuItemString:
				DrawTxt(img, item.ValueString, 78, y+offsetValues, tightpixel15.Font)
			case MenuItemTypeInt: // we now have Value (float) and ValueInt
				DrawTxtRight(img, strconv.Itoa(int(item.ValueInt)), 120, y+1+offsetValues, tightpixel15.Font)
			case MenuItemTypeFloat:
				v := strconv.FormatFloat(item.Value, 'f', 2, 64)
				DrawTxtRight(img, v, 120, y+1+offsetValues, tightpixel15.Font)
			case MenuItemTypeOnOff:
				DrawTxt(img, "on   off", 82, 13+offsetValues, tightpixel15.Font)
				top := 13 + offsetValues - 1
				bot := 13 + offsetValues + menuSpacingValues - 1
				switch item.On {
				case true:
					Line(img, 80, top, 80, bot)
					Line(img, 92, top, 92, bot)
				case false:
					Line(img, 102, top, 102, bot)
					Line(img, 117, top, 117, bot)
				}
			}
		}
	}

	// draw scroll bar if we have more than 4 items
	if count > 4 {
		sbHeight := 50
		sbWidth := 4
		x := 123
		y := 8
		Rect(img, x, y, sbWidth, sbHeight)
		screenCount := (count + itemsPerScreen - 1) / itemsPerScreen
		blockHeight := sbHeight / screenCount
		RectFilled(img, x, y+blockHeight*screen, sbWidth, blockHeight)
		// draw arrows
		if screen > 0 {
			Polyline(img, x, y, x+2, y-2, x+4, y)
		}

		if screen < (screenCount - 1) {
			Polyline(img,
				x, y+sbHeight,
				x+2, y+sbHeight+2,
				x+4, y+sbHeight)
		}
	}
}

// MenuSelection is returned when a new item is selected
type MenuSelection string

// GetArrowPos gets the arrow pos
func (m *Menu) GetArrowPos() int {
	return m.arrowPos
}

// ResetArrowPos resets the arrow pos
func (m *Menu) ResetArrowPos() {
	m.arrowPos = 0
}

// Key handles key input
func (m *Menu) Key(key isdata.Key) (ScreenID, interface{}, bool) {
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
			m.ResetArrowPos()
			return item.Screen, nil, true
		case MenuItemTypeSelect:
			return ScreenIDNoChange, MenuSelection(item.Description), true
		case MenuItemTypeOnOff, MenuItemTypeCommand:
			return ScreenIDNoChange, item.Message, true
		}
	}

	return ScreenIDNoChange, nil, true
}
