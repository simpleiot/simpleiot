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
	softKeys   *SoftKeys
	menu       *Menu
	state      *isdata.State
	config     *isdata.Config
	db         *isdb.IsDb
	dataLoaded bool
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

	// we don't want to load the db every time the db renders (0.5s) as
	// it may eventually be a long process so only do it once when the
	// screen is first displayed
	if !s.dataLoaded {
		// FIXME, not sure the loading is displaying -- probably
		// need to use a dialog instead
		Clear(img)
		Heading(img, "Loading History ...")
		s.menu.ResetItems()
		// extract faults from database
		faults, _ := s.db.ReadFaultHist()

		// display faults from most recent
		for i := len(faults) - 1; i >= 0; i-- {
			fault := faults[i]
			disp := isdata.SampleTypeToDisp(fault.Type)
			s.menu.AddItemStringLong(formatTime(fault.Time), disp)
		}

		s.dataLoaded = true

	}

	Clear(img)
	Heading(img, "Fault History")

	s.menu.Render(img)
	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *FaultsHistoryScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1: // Back
		s.dataLoaded = false
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
		return a(int(y), true) + "/" + a(int(m), true) + "/" + a(int(d), true)
	}
	h, m, s := t.Clock()
	return a(h, false) + ":" + a(m, false) + ":" + a(s, false)
}

func a(i int, date bool) string {
	a := strconv.Itoa(i)

	// for dates, if yyyy, make yy
	if date {
		if len(a) == 4 { // if the string is a yyyy
			a = a[2:] // make yy
		}

		// for times, add 0 in front of one digit values
	} else {

		if len(a) <= 1 {
			a = "0" + a
		}
	}
	return a
}
