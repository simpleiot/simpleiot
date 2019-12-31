package isui

import (
	"image/draw"
	"time"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

const (
	posYear int = iota
	posMonth
	posDay
	posHour
	posMin
)

// TimeEntryScreen is a customized textEntryScreen
type TimeEntryScreen struct {
	timeEdit time.Time
	position int
	softKeys *SoftKeys
}

// NewTimeEntryScreen returns a new time entry screen
func NewTimeEntryScreen() *TimeEntryScreen {
	return &TimeEntryScreen{
		softKeys: NewSoftKeys("done", "cancel"),
	}
}

// InitTimeEdit initializes the fields of timeEdit
func (s *TimeEntryScreen) InitTimeEdit() {
	s.timeEdit = time.Now()
}

//Render renders the text entry screen
func (s *TimeEntryScreen) Render(img draw.Image) {

	year, month, day := Date(s.timeEdit, true, true)
	hour, min, _ := Clock(s.timeEdit, true)
	timeStr := year + "/" + month + "/" + day + " " + hour + ":" + min

	// Header - time being edited
	txtStartX := DrawTxtCentered(img, timeStr, 64, 2, tightpixel15.Font) //assign margin, draw text
	width := 116
	Rect(img, 64-width/2-2, 0, width+2, 13)

	// cursor
	index := 3
	index = index + s.position*3
	widthString := tightpixel15.Font.MeasureString(timeStr[:index])
	Line(img, txtStartX+widthString, 11, txtStartX+widthString+4, 11)

	// soft keys
	s.softKeys.Render(img, 0, 54)
}

//Key handles some key inputs and passes the rest to inputChars
func (s *TimeEntryScreen) Key(key isdata.Key) TextEntryCommand {
	switch key {
	case isdata.KeySK1Release: // save
		return TextEntryCommandSave
	case isdata.KeySK2: // cancel
		return TextEntryCommandCancel
	case isdata.KeyEnter, isdata.KeyEnterHold, isdata.KeyRight, isdata.KeyRightHold:
		s.right()
	case isdata.KeyLeft, isdata.KeyLeftHold:
		s.left()
	case isdata.KeyUp, isdata.KeyUpHold:
		s.increment()
	case isdata.KeyDown, isdata.KeyDownHold:
		s.decrement()
	default:
		//log.Println("Time entry screen: unhandled key of type: ", key)
	}

	return TextEntryCommandNone
}

// GetTimeEdit returns the text that is being edited
func (s *TimeEntryScreen) GetTimeEdit() time.Time {
	return s.timeEdit
}

//ExitEdit resets the position and index
func (s *TimeEntryScreen) ExitEdit() {
	s.position = posYear
}

func (s *TimeEntryScreen) right() {
	s.position++
	if s.position > posMin {
		s.position = posMin
	}
}

func (s *TimeEntryScreen) left() {
	s.position--
	if s.position < posYear {
		s.position = posYear
	}
}

func (s *TimeEntryScreen) increment() {
	switch s.position {
	case posYear:
		s.timeEdit = s.timeEdit.AddDate(1, 0, 0)
	case posMonth:
		s.timeEdit = s.timeEdit.AddDate(0, 1, 0)
	case posDay:
		s.timeEdit = s.timeEdit.AddDate(0, 0, 1)
	case posHour:
		s.timeEdit = s.timeEdit.Add(time.Hour)
	case posMin:
		s.timeEdit = s.timeEdit.Add(time.Minute)
	}
}

func (s *TimeEntryScreen) decrement() {
	switch s.position {
	case posYear:
		s.timeEdit = s.timeEdit.AddDate(-1, 0, 0)
	case posMonth:
		s.timeEdit = s.timeEdit.AddDate(0, -1, 0)
	case posDay:
		s.timeEdit = s.timeEdit.AddDate(0, 0, -1)
	case posHour:
		s.timeEdit = s.timeEdit.Add(-1 * time.Hour)
	case posMin:
		s.timeEdit = s.timeEdit.Add(-1 * time.Minute)
	}
}
