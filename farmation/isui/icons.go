package isui

import (
	"fmt"
	"image/draw"
)

// Icons is a widget that renders icons on the home and status screens
type Icons struct {
	icons map[string]*iconFields
}

type iconFields struct {
	on      bool
	onIcon  string
	offIcon string
	x       int
	y       int
}

// NewIcons initializes the icons
func NewIcons() *Icons {
	ret := Icons{}
	// Add new icons
	margin := 110
	ret.icons = make(map[string]*iconFields)
	ret.icons["arm"] = &iconFields{onIcon: "arm.png", offIcon: "arm.png", x: margin, y: 1}
	ret.icons["pump"] = &iconFields{onIcon: "pump.png", offIcon: "pump.png", x: margin, y: 18}

	return &ret
}

// Set icon to on or off
func (i *Icons) Set(iconName string, on bool) {
	i.icons[iconName].on = on
}

// Render the widget
func (i *Icons) Render(img draw.Image) {

	// Draw all icons
	var icon string
	for _, iconFields := range i.icons {
		if iconFields.on {
			icon = iconFields.onIcon
		} else {
			icon = iconFields.offIcon
		}
		err := DrawPng(img, icon, iconFields.x, iconFields.y)
		if err != nil {
			s := fmt.Sprintf("error drawing %s: %s", icon, err)
			fmt.Println(s)
		}
	}
}
