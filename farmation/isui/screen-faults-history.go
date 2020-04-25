package isui

import (
	"image/draw"
	"log"
	"time"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/farmation/isdb"
)

// define state used for loading history data
const (
	hsStateEnter = iota
	hsStateLoading
	hsStateLoaded
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
	ch                chan []data.Sample
	hsState           int
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

func (s *FaultsHistoryScreen) storeFaults(faultsIn []data.Sample) {
	// need to reverse faults here
	s.faults = make([]data.Sample, len(faultsIn))
	fi := 0
	for i := len(faultsIn) - 1; i >= 0; i-- {
		s.faults[fi] = faultsIn[i]
		fi++
	}

	s.menu.ResetItems()

	// display faults from most recent
	for i := range s.faults {
		fault := s.faults[i]
		faultDisplay := isdata.SampleTypeToDisp(fault.Type)

		var timeDisplay string
		// if fault was more than 24 hrs ago, display date,
		// else display clock time
		if time.Since(fault.Time) >= time.Duration(24*time.Hour) {
			year, month, day := Date(fault.Time, false, false)
			timeDisplay = year + "/" + month + "/" + day
		} else {
			hour, min, sec := Clock(fault.Time, true)
			timeDisplay = hour + ":" + min + ":" + sec
		}

		s.menu.AddItemFaultHistory(timeDisplay, faultDisplay)
	}
}

// Render updates the home screen, and provides an image
func (s *FaultsHistoryScreen) Render(img draw.Image) {
	if s.displayDetails {
		s.faultsHistDetails.Render(img)
	} else {
		switch s.hsState {
		case hsStateEnter:
			s.ch = make(chan []data.Sample)
			go func(c chan []data.Sample) {
				start := time.Now().AddDate(0, 0, -7)
				faults, err := s.db.ReadFaultHist(start)
				log.Printf("displaying %v faults\n", len(faults))

				if err != nil {
					log.Println("Error reading faults: ", err)
				}
				c <- faults
			}(s.ch)

			Clear(img)
			Heading(img, "Loading History")
			DrawTxt(img, "This may take a bit", 8, 15, tightpixel15.Font)
			DrawTxt(img, "please wait ...", 8, 25, tightpixel15.Font)
			s.menu.ResetItems()
			s.hsState = hsStateLoading
		case hsStateLoading:
			select {
			case faults, ok := <-s.ch:
				if !ok {
					log.Println("Channel closed!")
				}
				s.storeFaults(faults)
				s.hsState = hsStateLoaded
			default:
				// no-op used to make channel non blocking
			}

		case hsStateLoaded:
			Clear(img)
			if len(s.faults) > 0 {
				Heading(img, "Fault history in past week")
			} else {
				Heading(img, "Yay, no faults in past week")
			}

			s.menu.Render(img)

		}
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
			s.hsState = hsStateEnter
			return ScreenIDHome, nil, true
		case isdata.KeySK1Release: // Back
			s.menu.ResetArrowPos() // return arrow to top of screen
			s.hsState = hsStateEnter
			return ScreenIDPrev, nil, true
		case isdata.KeyEnter, isdata.KeySK2: // Details
			if len(s.faults) >= 1 {
				s.displayDetails = true
				s.faultsHistDetails.fault = s.faults[s.menu.GetArrowPos()]
			}
		case isdata.KeyUp, isdata.KeyUpHold, isdata.KeyDown, isdata.KeyDownHold, isdata.KeyRight, isdata.KeyRightHold, isdata.KeyLeft, isdata.KeyLeftHold:
			return s.menu.Key(key)
		}
	}
	return ScreenIDNoChange, nil, true
}
