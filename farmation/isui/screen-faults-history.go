package isui

import (
	"image/draw"
	"strconv"
	"time"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/farmation/isdb"
)

// FaultsHistoryScreen is used to display fault history
type FaultsHistoryScreen struct {
	faults            []data.Sample
	softKeys          *SoftKeys
	menu              *Menu
	state             *isdata.State
	config            *isdata.Config
	db                *isdb.IsDb
	faultsHistDetails *FaultsHistDetailsScreen
	displayDetails    bool
	dataLoaded        bool
}

// NewFaultsHistoryScreen initializes and returns a screen
func NewFaultsHistoryScreen(state *isdata.State, config *isdata.Config, db *isdb.IsDb) *FaultsHistoryScreen {

	return &FaultsHistoryScreen{
		softKeys:          NewSoftKeys("back", "details"),
		menu:              NewMenu(),
		faultsHistDetails: NewFaultsHistDetailsScreen(state, config),
		state:             state,
		config:            config,
		db:                db,
	}
}

// Render updates the home screen, and provides an image
func (s *FaultsHistoryScreen) Render(img draw.Image) {
	if s.displayDetails {
		s.faultsHistDetails.Render(img)
	} else {
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
			s.faults, _ = s.db.ReadFaultHist()

			// display faults from most recent
			for i := len(s.faults) - 1; i >= 0; i-- {
				fault := s.faults[i]
				faultDisplay := isdata.SampleTypeToDisp(fault.Type)

				var timeDisplay string
				// if fault was more than 24 hrs ago, display date,
				// else display clock time
				if time.Since(fault.Time) >= time.Duration(24*time.Hour) {
					timeDisplay, _ = stringTime(fault.Time)
				} else {
					_, timeDisplay = stringTime(fault.Time)
				}

				s.menu.AddItemFaultHistory(timeDisplay, faultDisplay)
			}

			s.dataLoaded = true

		}

		Clear(img)
		Heading(img, "Fault History")

		s.menu.Render(img)
		s.softKeys.Render(img, 0, 54)
	}
}

// Key processes keypad input to this screen
func (s *FaultsHistoryScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	if s.displayDetails {
		switch key {
		case isdata.KeySK1: // Back
			s.displayDetails = false
		}
	} else {
		switch key {
		case isdata.KeySK1: // Back
			s.dataLoaded = false
			return ScreenIDPrev, nil, true
		case isdata.KeyEnter, isdata.KeySK2: // Details
			s.displayDetails = true
			s.faultsHistDetails.fault = s.faults[len(s.faults)-1-s.menu.GetArrowPos()]
		case isdata.KeyUp, isdata.KeyDown, isdata.KeyRight, isdata.KeyLeft:
			return s.menu.Key(key)
		}
	}
	return ScreenIDNoChange, nil, true
}

// formats a time into two strings, a date (form yy/mm/dd) and a clock time (form hh:mm:ss)
func stringTime(t time.Time) (string, string) {
	y, m, d := t.Date()
	yS, mS, dS := strconv.Itoa(int(y)), strconv.Itoa(int(m)), strconv.Itoa(int(d))
	// if yyyy, make yy
	if len(yS) > 2 {
		yS = yS[2:]
	}

	h, min, s := t.Clock()
	hS, minS, sS := addZero(strconv.Itoa(h)), addZero(strconv.Itoa(min)), addZero(strconv.Itoa(s))

	return yS + "/" + mS + "/" + dS,
		addZero(hS) + ":" + addZero(minS) + ":" + addZero(sS)
}

// add 0 in front of one digit values
func addZero(a string) string {
	if len(a) <= 1 {
		a = "0" + a
	}
	return a
}
