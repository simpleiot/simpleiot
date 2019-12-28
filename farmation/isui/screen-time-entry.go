package isui

import (
	"image/draw"
	"strconv"
	"time"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

type position int

const (
	posYear position = iota
	posMonth
	posDay
	posHour
	posMin
)

// TimeEntryScreen is a customized textEntryScreen
type TimeEntryScreen struct {
	timeEdit [5]string
	position position
	index    int
	softKeys *SoftKeys
}

// NewTimeEntryScreen returns a new time entry screen
func NewTimeEntryScreen() *TimeEntryScreen {
	return &TimeEntryScreen{
		index:    2, // start index at third digit of year
		softKeys: NewSoftKeys("done", "cancel"),
	}
}

// InitTimeEdit initializes the fields of timeEdit
func (s *TimeEntryScreen) InitTimeEdit() {

	s.index = 2 // start at 3rd digit of year

	year, month, day := Date(time.Now(), true)
	hour, min, _ := Clock(time.Now())

	s.timeEdit[posYear] = year
	s.timeEdit[posMonth] = month
	s.timeEdit[posDay] = day
	s.timeEdit[posHour] = hour
	s.timeEdit[posMin] = min
}

//Render renders the text entry screen
func (s *TimeEntryScreen) Render(img draw.Image) {

	time := s.timeEdit[posYear] + "/" + s.timeEdit[posMonth] + "/" + s.timeEdit[posDay] + " " + s.timeEdit[posHour] + ":" + s.timeEdit[posMin]
	// Header - time being edited
	txtStartX := DrawTxtCentered(img, time, 64, 2, tightpixel15.Font) //assign margin, draw text
	width := 116
	Rect(img, 64-width/2-2, 0, width+2, 13)

	// cursor
	var widthString int
	var dividerChar string
	for i, str := range s.timeEdit[:s.position] {
		switch i {
		case int(posDay):
			dividerChar = " "
		case int(posHour):
			dividerChar = ":"
		default:
			dividerChar = "/"
		}
		widthString = widthString + tightpixel15.Font.MeasureString(str+dividerChar)
	}
	widthString = widthString + tightpixel15.Font.MeasureString(s.timeEdit[s.position][:s.index])

	_, widthChar := tightpixel15.Font.MeasureRune(rune(s.timeEdit[s.position][s.index]))
	Line(img, txtStartX+widthString+widthChar/2-2, 11, txtStartX+widthString+widthChar/2+2, 11)

	// soft keys
	s.softKeys.Render(img, 0, 54)
}

//Key handles some key inputs and passes the rest to inputChars
func (s *TimeEntryScreen) Key(key isdata.Key) TextEntryCommand {
	switch key {
	case isdata.KeySK1Release: // save
		s.applyLimits()
		return TextEntryCommandSave
	case isdata.KeySK2: // cancel
		s.applyLimits()
		return TextEntryCommandCancel
	case isdata.KeyEnter:
		s.right()
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

	year, _ := strconv.Atoi(s.timeEdit[posYear])
	month, _ := strconv.Atoi(s.timeEdit[posMonth])
	day, _ := strconv.Atoi(s.timeEdit[posDay])
	hour, _ := strconv.Atoi(s.timeEdit[posHour])
	min, _ := strconv.Atoi(s.timeEdit[posMin])

	// FIXME use correct timezone
	return time.Date(year, time.Month(month), day, hour, min, 0, 0, time.UTC)
}

//ExitEdit resets the position and index
func (s *TimeEntryScreen) ExitEdit() {
	s.position = posYear
	s.index = 0
}

func (s *TimeEntryScreen) right() {
	s.index++
	if s.index >= len(s.timeEdit[s.position]) {

		// check to make sure the edited value is within upper limit
		s.applyLimits()

		s.index = 0
		s.position++
		if s.position > posMin {
			s.position = posYear
			// start index at 3rd digit in year
			// don't allow user to edit first two
			s.index = 2
		}

	}
}

func (s *TimeEntryScreen) increment() {
	byteEdit := []byte(s.timeEdit[s.position]) // convert to a slice of bytes to replace characters
	v := int(s.timeEdit[s.position][s.index])
	v++
	if v > 57 {
		v = 48
	}
	byteEdit[s.index] = byte(v)
	s.timeEdit[s.position] = string(byteEdit)
}

func (s *TimeEntryScreen) decrement() {
	byteEdit := []byte(s.timeEdit[s.position])
	v := int(s.timeEdit[s.position][s.index])
	v--
	if v < 48 {
		v = 57
	}
	byteEdit[s.index] = byte(v)
	s.timeEdit[s.position] = string(byteEdit)
}

func (s *TimeEntryScreen) applyLimits() {
	value, _ := strconv.Atoi(s.timeEdit[s.position])
	switch s.position {
	case posMonth:
		if value > 12 {
			s.timeEdit[s.position] = strconv.Itoa(12)
		}
	case posDay:
		// FIXME adjust for month
		if value > 31 {
			s.timeEdit[s.position] = strconv.Itoa(31)
		}
	case posHour:
		if value > 24 {
			s.timeEdit[s.position] = strconv.Itoa(24)
		}
	case posMin:
		if value > 59 {
			s.timeEdit[s.position] = strconv.Itoa(59)
		}
	}
}
