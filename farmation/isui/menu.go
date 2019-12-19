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
	MenuItemString
	MenuItemTypeFaultHistory
	MenuItemStringRight
	MenuItemStringIP
	MenuItemTypeStringDown
	MenuItemTypeInt
	MenuItemTypeFloat
	MenuItemTypeOnOff
	MenuItemTypeAutoOffOn
	MenuItemTypeSelect
	MenuItemTypeCommand
	MenuItemTypeBreak
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
func NewMenu(selectMenu bool, selectedIndex int) *Menu {
	if selectMenu {
		return &Menu{
			arrowPos: selectedIndex,
		}
	}

	return &Menu{}
}

func (m *Menu) updateShowValues() {
	for _, item := range m.items {
		if item.Type != MenuItemTypeScreen && item.Type != MenuItemTypeSelect && item.Type != MenuItemTypeBreak {
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

// AddItemFaultHistory is specialized for this application:
// Longer value, no box
func (m *Menu) AddItemFaultHistory(desc string, value string) {
	m.items = append(m.items, MenuItem{
		Description: desc,
		ValueString: value,
		Type:        MenuItemTypeFaultHistory,
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

// AddItemStringIP is customized for displaying an IP address
// The difference from AddItemStringDown is that this type is truncated from the beginning
// of the string instead of the end, displaying the last part of the string.
func (m *Menu) AddItemStringIP(desc string, value string) {
	m.items = append(m.items, MenuItem{
		Description: desc,
		ValueString: value,
		Type:        MenuItemStringIP,
	})

	m.updateShowValues()
}

// AddItemStringDown adds a string item with the value rendered one pixel futhur down than
// AddItemString
func (m *Menu) AddItemStringDown(desc string, value string) {
	m.items = append(m.items, MenuItem{
		Description: desc,
		Type:        MenuItemTypeStringDown,
		ValueString: value,
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
func (m *Menu) AddItemCommand(desc, command string, msg interface{}) {
	m.items = append(m.items, MenuItem{
		Description: desc,
		Type:        MenuItemTypeCommand,
		Message:     msg,
		ValueString: command,
	})

	m.updateShowValues()
}

// AddItemBreak adds a menu break item
func (m *Menu) AddItemBreak(desc string) {
	m.items = append(m.items, MenuItem{
		Description: desc,
		Type:        MenuItemTypeBreak,
	})

	m.updateShowValues()
}

// Render is used to draw a list of params, handles scrolling, etc.
func (m *Menu) Render(img draw.Image) {

	var menuSpacingValues = 11
	var menuSpacingText = 11
	if len(m.items) > 0 {
		if m.items[0].Type == MenuItemTypeFaultHistory || !m.showValues {
			menuSpacingValues = 10
			menuSpacingText = 10
		}
	}

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

	if len(m.items) > 0 {
		if !m.showValues {
			x = 30
			Arrow(img, x-8, 15+arrowScreenPos*menuSpacingText)
		} else {
			switch m.items[m.arrowPos].Type {
			case MenuItemTypeFaultHistory:
				Arrow(img, x+37, 15+arrowScreenPos*menuSpacingValues)
			case MenuItemTypeBreak:
			default:
				Arrow(img, x+65, 15+arrowScreenPos*menuSpacingValues)
			}
		}
	}

	y = 11
	for i := start; i <= end; i++ {
		screenIndex := i - start
		item := m.items[i]
		offsetText := screenIndex * menuSpacingText
		offsetValues := screenIndex * menuSpacingValues
		if item.Type != MenuItemTypeBreak {
			DrawTxt(img, item.Description, x, y+offsetText, tightpixel15.Font)
		}

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
				item.Type != MenuItemTypeFaultHistory &&
				item.Type != MenuItemTypeBreak {
				Rect(img, 76, 10+offsetValues, 45, menuSpacingValues)
			}
			switch item.Type {
			case MenuItemTypeScreen:
				DrawTxt(img, "open", 78, y+offsetValues, tightpixel15.Font)
			case MenuItemTypeSelect:
				DrawTxt(img, "select", 78, y+offsetValues, tightpixel15.Font)
			case MenuItemTypeCommand:
				v := truncateMenuVal(item.ValueString)
				DrawTxt(img, v, 78, y+offsetValues, tightpixel15.Font)
			case MenuItemString:
				v := truncateMenuVal(item.ValueString)
				DrawTxt(img, v, 78, y+offsetValues, tightpixel15.Font)
			case MenuItemTypeFaultHistory:
				v := truncateMenuVal(item.ValueString)
				DrawTxt(img, v, 49, y+offsetValues, tightpixel15.Font)
			case MenuItemStringRight:
				v := truncateMenuVal(item.ValueString)
				DrawTxtRight(img, v, 120, y+1+offsetValues, tightpixel15.Font)
			case MenuItemStringIP:
				v := truncateMenuValBeginning(item.ValueString)
				DrawTxt(img, v, 78, y+1+offsetValues, tightpixel15.Font)
			case MenuItemTypeStringDown:
				v := truncateMenuVal(item.ValueString)
				DrawTxt(img, v, 78, y+1+offsetValues, tightpixel15.Font)
			case MenuItemTypeInt: // we now have Value (float) and ValueInt
				v := truncateMenuVal(strconv.Itoa(int(item.ValueInt)))
				DrawTxtRight(img, v, 120, y+1+offsetValues, tightpixel15.Font)
			case MenuItemTypeFloat:
				v := truncateMenuVal(strconv.FormatFloat(item.Value, 'f', 2, 64))
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
			case MenuItemTypeBreak:
				MenuBreak(img, item.Description, y+offsetValues)
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

func truncateMenuVal(v string) string {
	for i := len(v) - 1; i >= 0; i-- {
		if tightpixel15.Font.MeasureString(v) <= 41 { // if the value will fit in a menu box
			return v
		}
		v = v[:len(v)-1]
	}
	return ""
}

func truncateMenuValBeginning(v string) string {
	splitPoint := 0
	for i := 0; i <= len(v)-1; i++ {
		if tightpixel15.Font.MeasureString(v[splitPoint:]) <= 41 { // if the value will fit in a menu box
			break
		}
		splitPoint++
	}
	return v[splitPoint:]
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
	if m.items[0].Type == MenuItemTypeSelect {
		for index, item := range m.items {
			if item.Selected {
				m.arrowPos = index
			}
		}
	}
}

// Key handles key input
func (m *Menu) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeyUp, isdata.KeyUpHold:
		m.arrowUp()
		item := m.items[m.arrowPos]
		// skip past menu break positions
		if item.Type == MenuItemTypeBreak {
			m.arrowUp()
		}
	case isdata.KeyDown, isdata.KeyDownHold:
		if len(m.items) > 0 {
			m.arrowDown()
			item := m.items[m.arrowPos]
			// skip past menu break positions
			if item.Type == MenuItemTypeBreak {
				m.arrowDown()
			}
		}
	case isdata.KeyEnter:
		if len(m.items) <= 0 {
			return ScreenIDNoChange, nil, true
		}

		item := m.items[m.arrowPos]
		switch item.Type {
		case MenuItemTypeScreen, MenuItemTypeFaultHistory:
			return item.Screen, nil, true
		case MenuItemTypeSelect, MenuItemTypeOnOff, MenuItemTypeAutoOffOn, MenuItemTypeCommand:
			return ScreenIDNoChange, item.Message, true
		}
	}

	return ScreenIDNoChange, nil, true
}

func (m *Menu) arrowUp() {
	m.arrowPos--
	if m.arrowPos < 0 {
		m.arrowPos = 0
	}
}

func (m *Menu) arrowDown() {
	m.arrowPos++
	if m.arrowPos >= len(m.items) {
		m.arrowPos = len(m.items) - 1
	}
}
