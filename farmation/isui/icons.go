package isui

import (
	"fmt"
	"image/draw"
	"io"
)

// Icons is a widget that renders icons on the home and status screens
type Icons struct {
	icons map[string]*iconFields
}

type iconFields struct {
	on      bool
	iconOn  string
	iconOff string
	page    int
	icon1   string
	icon2   string
	icon3   string
	icon4   string
	x       int
	y       int
}

const (
	home int = iota
	status1
	status2
	status3
)

// NewIcons initializes the icons
func NewIcons(pageInd, inputs, outputs bool) *Icons {
	ret := Icons{}
	// Add new icons
	marginCenter, marginRight, marginLeft := 59, 115, 1
	ret.icons = make(map[string]*iconFields)
	if pageInd {
		ret.icons["page indicator"] = &iconFields{icon1: "indicator-home.png", icon2: "indicator-status1.png", icon3: "indicator-status2.png", icon4: "indicator-status3.png", x: marginCenter, y: 1}
	}
	if inputs {
		ret.icons["pump in"] = &iconFields{iconOn: "pump.png", iconOff: "", x: marginLeft, y: 4}
		ret.icons["water"] = &iconFields{iconOn: "water-on.png", iconOff: "", x: marginLeft + 2, y: 20}
		ret.icons["irrigator"] = &iconFields{iconOn: "irrigator.png", iconOff: "", x: marginLeft, y: 40}
	}
	if outputs {
		ret.icons["arm"] = &iconFields{iconOn: "arm.png", iconOff: "", x: marginRight - 1, y: 2}
		ret.icons["pump"] = &iconFields{iconOn: "pump.png", iconOff: "", x: marginRight, y: 22}
		ret.icons["shutdown"] = &iconFields{iconOn: "shutdown.png", iconOff: "", x: marginRight, y: 38}
	}

	return &ret
}

// SetOnOff sets an icon to on or off
func (i *Icons) SetOnOff(iconName string, on bool) {
	i.icons[iconName].on = on
}

// SetPage sets a page indicator icon
func (i *Icons) SetPage(iconName string, page int) {
	i.icons[iconName].page = page
}

// Render the widget
func (i *Icons) Render(img draw.Image) {

	// Draw all icons
	var icon string
	for _, iconFields := range i.icons {

		// Which icon to render for on/off icons
		if iconFields.on {
			icon = iconFields.iconOn
		} else {
			icon = iconFields.iconOff
		}

		if iconFields.icon1 != "" { // if the icon is a page position indicator
			// Which icon to render for status pages position icon
			switch iconFields.page {
			case home:
				icon = iconFields.icon1
			case status1:
				icon = iconFields.icon2
			case status2:
				icon = iconFields.icon3
			case status3:
				icon = iconFields.icon4
			}
		}

		// draw icon
		err := DrawPng(img, icon, iconFields.x, iconFields.y)
		if err != nil && err != io.ErrUnexpectedEOF {
			s := fmt.Sprintf("error drawing %s: %s", icon, err)
			fmt.Println(s)
		}
	}
}
