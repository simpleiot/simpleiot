package isui

import (
	"image/draw"
	"log"
	"strconv"
	"time"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/farmation/isdb"
)

const (
	ssStateNormal = iota
	ssStateConfirm
	ssStateDeleting
)

// StorageScreen displays storage stats and offers data purge option
type StorageScreen struct {
	softKeys              *SoftKeys
	state                 *isdata.State
	config                *isdata.Config
	dbData                *isdb.IsDb
	menu                  Menu
	ch                    chan error
	sampleCount           int
	lastSampleCountUpdate time.Time
	ssState               int
}

// NewStorageScreen creates a modem information screen
func NewStorageScreen(state *isdata.State, config *isdata.Config, dbData *isdb.IsDb) *StorageScreen {
	isdata.InitState(state) // make sure that ProductStates and FieldStates arrays are large enough
	menu := Menu{}

	return &StorageScreen{
		// update this from sample screen
		state:  state,
		config: config,
		menu:   menu,
		dbData: dbData,
	}
}

// Render updates the screen, and provides an image
func (s *StorageScreen) Render(img draw.Image) {
	Clear(img)
	Heading(img, "Data Storage Admin")

	switch s.ssState {
	case ssStateConfirm:
		DrawTxt(img, "Are you sure you want", 8, 15, tightpixel15.Font)
		DrawTxt(img, "to delete all history data?", 8, 25, tightpixel15.Font)
		s.softKeys = NewSoftKeys("back", "yes", "no")
	case ssStateDeleting:
		s.softKeys = NewSoftKeys("back")
		select {
		case err, _ := <-s.ch:
			// delete operation is finished, process results
			if err != nil {
				log.Println("Error deleting data: ", err)
				DrawTxt(img, "Error deleting data", 8, 15, tightpixel15.Font)
			} else {
				DrawTxt(img, "Data deleted", 8, 15, tightpixel15.Font)
			}
			s.ssState = ssStateNormal
			s.lastSampleCountUpdate = time.Time{}
		default:
			// causes above channel read not to block
			DrawTxt(img, "Deleting data, please wait ...", 8, 15, tightpixel15.Font)
		}

	default:
		s.softKeys = NewSoftKeys("back", "purge")
		if time.Since(s.lastSampleCountUpdate) > time.Second {
			var err error
			s.sampleCount, err = s.dbData.GetSampleCount()
			if err != nil {
				log.Println("Error getting sample count")
			}
			s.lastSampleCountUpdate = time.Now()
		}

		s.menu.ResetItems()
		s.menu.AddItemStringRight("Record Count", strconv.Itoa(s.sampleCount))
		s.menu.AddItemStringRight("Data Usage", strconv.FormatFloat(s.state.DataUsage, 'f', 0, 64)+"%")
		s.menu.AddItemStringRight("Rootfs Usage", strconv.FormatFloat(s.state.RootUsage, 'f', 0, 64)+"%")
		s.menu.Render(img)

	}

	s.softKeys.Render(img, 0, 54)
}

// Key processes keypad input to this screen
func (s *StorageScreen) Key(key isdata.Key) (ScreenID, interface{}, bool) {
	switch key {
	case isdata.KeySK1Hold: // Back key held -> Home screen
		s.menu.ResetArrowPos() // return arrow to top of screen
		if s.ssState == ssStateConfirm {
			s.ssState = ssStateNormal
		}
		return ScreenIDHome, nil, true
	case isdata.KeySK1Release: // Back
		s.menu.ResetArrowPos() // return arrow to top of screen
		if s.ssState == ssStateConfirm {
			s.ssState = ssStateNormal
		}
		return ScreenIDPrev, nil, true
	case isdata.KeySK2: // purge for ssStateNormal, and Yes for ssStateConfirm
		if s.ssState == ssStateNormal {
			s.ssState = ssStateConfirm
		} else if s.ssState == ssStateConfirm {
			s.ch = make(chan error)
			go func(ch chan error) {
				start := time.Now()
				ch <- s.dbData.DeleteSamples()
				log.Printf("Took %v to delete data\n", time.Since(start))
			}(s.ch)
			s.ssState = ssStateDeleting
		}
	case isdata.KeySK3: // No for ssStateConfirm
		if s.ssState == ssStateConfirm {
			s.ssState = ssStateNormal
		}
	case isdata.KeyUp, isdata.KeyUpHold, isdata.KeyDown, isdata.KeyDownHold, isdata.KeyRight, isdata.KeyRightHold, isdata.KeyLeft, isdata.KeyLeftHold, isdata.KeyEnter, isdata.KeyEnterHold:
		return s.menu.Key(key)
	}

	return ScreenIDNoChange, nil, true
}
