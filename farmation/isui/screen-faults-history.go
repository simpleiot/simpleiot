package isui

import (
	"image/draw"
	"strconv"
	"time"

	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/farmation/isdb"
)

// FaultsHistoryScreen is used to display fault history
type FaultsHistoryScreen struct {
	softKeys *SoftKeys
	menu     *Menu
	state    *isdata.State
	config   *isdata.Config
	db       *isdb.IsDb
}

// NewFaultsHistoryScreen initializes and returns a screen
func NewFaultsHistoryScreen(state *isdata.State, config *isdata.Config, db *isdb.IsDb) *FaultsHistoryScreen {

	return &FaultsHistoryScreen{
		softKeys: NewSoftKeys("back"),
		menu:     NewMenu(),
		state:    state,
		config:   config,
		db:       db,
	}
}

// Render updates the home screen, and provides an image
func (s *FaultsHistoryScreen) Render(img draw.Image) {
	Clear(img)
	s.menu.ResetItems()

	faults := s.db.ReadFaultHist(0, faults)

	for i, v := range s.state.FaultsHist {
		s.menu.AddItemStringLong(formatTime(s.state.FaultsHistTimes[i]), v)
	}

	Heading(img, "Fault History")
	s.menu.Render(img)
	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *FaultsHistoryScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1: // Back
		return ScreenIDPrev, nil, true
	case isdata.KeyUp, isdata.KeyDown, isdata.KeyRight, isdata.KeyLeft, isdata.KeyEnter:
		return s.menu.Key(key)
	}

	return ScreenIDNoChange, nil, true
}

// formats a time into a string:
// this is a clock time (form hh:mm:ss) if time.Now()
// is less than 24 hrs away and a date (form yy/mm/dd) otherwise
func formatTime(t time.Time) string {

	if time.Since(t) >= time.Duration(24*time.Hour) {
		y, m, d := t.Date()
		return a(int(y)) + "/" + a(int(m)) + "/" + a(int(d))
	}
	h, m, s := t.Clock()
	return a(h) + ":" + a(m) + ":" + a(s)
}

func a(i int) string {
	a := strconv.Itoa(i)
	if len(a) <= 1 {
		a = "0" + a
	}
	return a
}
