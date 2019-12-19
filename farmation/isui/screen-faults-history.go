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
		menu:              &Menu{},
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
					timeDisplay = date(fault.Time, false)
				} else {
					timeDisplay = clockTime(fault.Time)
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
		case isdata.KeySK1Hold: // Back key held -> Home screen
			s.menu.ResetArrowPos() // return arrow to top of screen
			return ScreenIDHome, nil, true
		case isdata.KeySK1Release: // Back
			s.displayDetails = false
		}
	} else {
		switch key {
		case isdata.KeySK1Hold: // Back key held -> Home screen
			s.menu.ResetArrowPos() // return arrow to top of screen
			s.dataLoaded = false
			return ScreenIDHome, nil, true
		case isdata.KeySK1Release: // Back
			s.menu.ResetArrowPos() // return arrow to top of screen
			s.dataLoaded = false
			return ScreenIDPrev, nil, true
		case isdata.KeyEnter, isdata.KeySK2: // Details
			if len(s.faults) >= 1 {
				s.displayDetails = true
				s.faultsHistDetails.fault = s.faults[len(s.faults)-1-s.menu.GetArrowPos()]
			}
		case isdata.KeyUp, isdata.KeyUpHold, isdata.KeyDown, isdata.KeyDownHold, isdata.KeyRight, isdata.KeyRightHold, isdata.KeyLeft, isdata.KeyLeftHold:
			return s.menu.Key(key)
		}
	}
	return ScreenIDNoChange, nil, true
}

// formats a time into a date string:
// yyyy/mm/dd if fullYear is true and yy/mm/dd otherwise
func date(t time.Time, fullYear bool) string {
	y, m, d := t.Date()
	yS, mS, dS := strconv.Itoa(int(y)), strconv.Itoa(int(m)), strconv.Itoa(int(d))

	if !fullYear {
		// if yyyy, make yy
		if len(yS) > 2 {
			yS = yS[2:]
		}
	}

	return yS + "/" + mS + "/" + dS
}

// formats a time into a clock time: hh:mm:ss
func clockTime(t time.Time) string {
	h, m, s := t.Clock()
	hS, mS, sS := addZero(strconv.Itoa(h)), addZero(strconv.Itoa(m)), addZero(strconv.Itoa(s))
	return addZero(hS) + ":" + addZero(mS) + ":" + addZero(sS)
}

// add 0 in front of one digit values
func addZero(a string) string {
	if len(a) <= 1 {
		a = "0" + a
	}
	return a
}
