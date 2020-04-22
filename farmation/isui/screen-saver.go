package isui

import (
	"image/draw"
	"time"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// ScreenSaver displays moving logo on screen
type ScreenSaver struct {
	logoPosX     int
	logoPosY     int
	timeLastJump time.Time
}

// NewScreenSaver returns a new screen saver screen
func NewScreenSaver() *ScreenSaver {
	return &ScreenSaver{}
}

// Render updates the home screen, and provides an image
func (s *ScreenSaver) Render(img draw.Image) {
	if time.Since(s.timeLastJump) > time.Second {
		Clear(img)
		DrawPng(img, "IS_logo_injector.png", s.logoPosX, s.logoPosY)
		s.logoPosX += 10

		if s.logoPosX > 120 {
			s.logoPosY += 5
			s.logoPosX = -60
		}

		if s.logoPosY > 30 {
			s.logoPosY = 0
		}

		s.timeLastJump = time.Now()
	}
}

// Key processes keypad input to this screen
func (s *ScreenSaver) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	return ScreenIDHome, nil, true
}
