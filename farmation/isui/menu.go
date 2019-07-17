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
	MenuItemStringLong
	MenuItemStringRight
	MenuItemTypeInt
	MenuItemTypeFloat
	MenuItemTypeOnOff
	MenuItemTypeAutoOffOn
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
	AutoOffOn   isdata.RelayControlStateType
	Precision   int
	Message     interface{}
	Selected    bool
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
func (m *Menu) AddItemSelect(desc string, msg interface{}, selected bool) {
	m.items = append(m.items, MenuItem{
		Description: desc,
		Type:        MenuItemTypeSelect,
		Message:     msg,
		Selected:    selected,
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
// Used for on/off and auto/off/on switches
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

// AddItemAutoOffOn adds a auto/off/on selection for relay control in
// the Diagnostics and Config screen
func (m *Menu) AddItemAutoOffOn(desc string, autoOffOn isdata.RelayControlStateType, msg interface{}) {
	m.items = append(m.items, MenuItem{
		Description: desc,
		Type:        MenuItemTypeAutoOffOn,
		AutoOffOn:   autoOffOn,
		Message:     msg,
	})

	m.updateShowValues()
}

// AddItemString adds a string item
func (m *Menu) AddItemString(desc string, value string) {
	m.items = append(m.items, MenuItem{
		Description: desc,
		ValueString: value,
		Type:        MenuItemString,
	})

	m.updateShowValues()
}

// AddItemStringLong adds a long string to the menu
// makes enough room by moving over arrow and lengthening
// rect
func (m *Menu) AddItemStringLong(desc string, value string) {
	m.items = append(m.items, MenuItem{
		Description: desc,
		ValueString: value,
		Type:        MenuItemStringLong,
	})

	m.updateShowValues()
}

// AddItemStringRight adds a string item with value rendered right-justified in rectagle
// one usage is to draw a number with a symbol as a string
func (m *Menu) AddItemStringRight(desc string, value string) {
	m.items = append(m.items, MenuItem{
		Description: desc,
		ValueString: value,
		Type:        MenuItemStringRight,
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

// AddItemFloat adds a float item to menu
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
		Arrow(img, x-8, 15+arrowScreenPos*menuSpacingText)
	} else {
		if m.items[0].Type != MenuItemStringLong {
			Arrow(img, x+65, 15+arrowScreenPos*menuSpacingValues)
		} else { // arrow needs moved over for long string
			Arrow(img, x+35, 15+arrowScreenPos*menuSpacingValues)
		}
	}

	y = 11
	for i := start; i <= end; i++ {
		screenIndex := i - start
		item := m.items[i]
		offsetText := screenIndex * menuSpacingText
		offsetValues := screenIndex * menuSpacingValues
		DrawTxt(img, item.Description, x, y+offsetText, tightpixel15.Font)

		if !m.showValues {
			switch item.Type {
			case MenuItemTypeSelect:
				if item.Selected {

					DrawTxtRevLarge(img, item.Description, x, y+offsetText, tightpixel15.Font)

				}
			}
		}

		if m.showValues {
			// draw values
			if item.Type != MenuItemTypeAutoOffOn && // auto/off/on needs slightly wider rect
				item.Type != MenuItemStringLong {
				Rect(img, 76, 10+offsetValues, 45, menuSpacingValues)
			}
			switch item.Type {
			case MenuItemTypeScreen:
				DrawTxt(img, "open", 78, y+offsetValues, tightpixel15.Font)
			case MenuItemTypeSelect:
				DrawTxt(img, "select", 78, y+offsetValues, tightpixel15.Font)
			case MenuItemTypeCommand:
				DrawTxt(img, "start", 78, y+offsetValues, tightpixel15.Font)
			case MenuItemString:
				DrawTxt(img, item.ValueString, 78, y+offsetValues, tightpixel15.Font)
			case MenuItemStringLong:
				DrawTxt(img, item.ValueString, 47, y+offsetValues, tightpixel15.Font)
			case MenuItemStringRight:
				DrawTxtRight(img, item.ValueString, 120, y+1+offsetValues, tightpixel15.Font)
			case MenuItemTypeInt: // we now have Value (float) and ValueInt
				DrawTxtRight(img, strconv.Itoa(int(item.ValueInt)), 120, y+1+offsetValues, tightpixel15.Font)
			case MenuItemTypeFloat:
				v := strconv.FormatFloat(item.Value, 'f', 2, 64)
				DrawTxtRight(img, v, 120, y+1+offsetValues, tightpixel15.Font)
			case MenuItemTypeOnOff:
				yShift := 11
				DrawTxt(img, "on   off", 82, yShift+offsetValues, tightpixel15.Font)
				switch item.On {
				case true:
					RectFilled(img, 79, yShift+offsetValues, tightpixel15.Font.MeasureString("on")+3, 10)
					DrawTxtRev(img, "on", 81, yShift+offsetValues, tightpixel15.Font)
				case false:
					RectFilled(img, 79+tightpixel15.Font.MeasureString("on")+13, yShift+offsetValues, tightpixel15.Font.MeasureString("off")+3, 10)
					DrawTxtRev(img, "off", 81+tightpixel15.Font.MeasureString("on")+13, yShift+offsetValues, tightpixel15.Font)
				}
			case MenuItemTypeAutoOffOn:
				yShift := 11
				Rect(img, 76, 10+offsetValues, 47, menuSpacingValues)
				DrawTxt(img, "auto", 78, yShift+offsetValues, tightpixel15.Font)
				DrawTxt(img, "off", 78+tightpixel15.Font.MeasureString("auto")+2, yShift+offsetValues, tightpixel15.Font)
				DrawTxt(img, "on", 78+tightpixel15.Font.MeasureString("autooff")+4, yShift+offsetValues, tightpixel15.Font)
				switch item.AutoOffOn {
				case 0:
					DrawTxtRev(img, "auto", 78, yShift+offsetValues, tightpixel15.Font)
				case 1:
					DrawTxtRev(img, "off", 78+tightpixel15.Font.MeasureString("auto")+2, yShift+offsetValues, tightpixel15.Font)
				case 2:
					DrawTxtRev(img, "on", 78+tightpixel15.Font.MeasureString("autooff")+4, yShift+offsetValues, tightpixel15.Font)
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
			return item.Screen, nil, true
		case MenuItemTypeSelect, MenuItemTypeOnOff, MenuItemTypeAutoOffOn, MenuItemTypeCommand:
			return ScreenIDNoChange, item.Message, true
		}
	}

	return ScreenIDNoChange, nil, true
}
