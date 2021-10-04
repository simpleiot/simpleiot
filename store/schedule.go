package store

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"
)

type schedule struct {
	startTime string
	endTime   string
	weekdays  []time.Weekday
}

func newSchedule(start, end string, weekdays []time.Weekday) *schedule {
	return &schedule{
		startTime: start,
		endTime:   end,
		weekdays:  weekdays,
	}
}

func (s *schedule) activeForTime(t time.Time) (bool, error) {
	tUTC := t.UTC()

	// parse out hour/minute
	matches := reHourMin.FindStringSubmatch(s.startTime)
	if len(matches) < 3 {
		return false, fmt.Errorf("TimeRange: invalid start: %v ", s.startTime)
	}

	startHour, err := strconv.Atoi(matches[1])
	if err != nil {
		return false, fmt.Errorf("TimeRange: error parsing start hour: %v", matches[1])
	}

	startMin, err := strconv.Atoi(matches[2])
	if err != nil {
		return false, fmt.Errorf("TimeRange: error parsing start hour: %v", matches[1])
	}

	matches = reHourMin.FindStringSubmatch(s.endTime)
	if len(matches) < 3 {
		return false, fmt.Errorf("TimeRange: invalid end: %v ", s.endTime)
	}

	endHour, err := strconv.Atoi(matches[1])

	if err != nil {
		return false, fmt.Errorf("TimeRange: error parsing end hour: %v", matches[1])
	}

	endMin, err := strconv.Atoi(matches[2])

	if err != nil {
		return false, fmt.Errorf("TimeRange: error parsing end hour: %v", matches[1])
	}

	y := tUTC.Year()
	m := tUTC.Month()
	d := tUTC.Day()

	start := time.Date(y, m, d, startHour, startMin, 0, 0, time.UTC)
	end := time.Date(y, m, d, endHour, endMin, 0, 0, time.UTC)

	timeRanges := timeRanges{
		{start, end},
	}

	if !timeRanges[0].end.After(timeRanges[0].start) {
		timeRanges[0].end = timeRanges[0].end.AddDate(0, 0, 1)

		timeRanges = append(timeRanges,
			timeRange{timeRanges[0].start.AddDate(0, 0, -1),
				timeRanges[0].end.AddDate(0, 0, -1),
			})
	}

	timeRanges.filterWeekdays(s.weekdays)

	if timeRanges.in(t) {
		return true, nil
	}

	return false, nil
}

var reHourMin = regexp.MustCompile(`(\d{1,2}):(\d\d)`)
var reDate = regexp.MustCompile(`(\d{4})-(\d{1,2})-(\d{1,2})`)

type timeRange struct {
	start time.Time
	end   time.Time
}

// in returns true if date is in time range
func (tr *timeRange) in(t time.Time) bool {
	if tr.start.After(tr.end) {
		log.Println("BUG: LocalTimeRange.In -- start is before end")
		return false
	}

	// normal situation
	if t.Before(tr.start) {
		return false
	}

	if t.Before(tr.end) {
		return true
	}

	return false
}

type timeRanges []timeRange

// in returns true if time is in any of the time ranges
func (trs *timeRanges) in(t time.Time) bool {
	for _, tr := range *trs {
		if tr.in(t) {
			return true
		}
	}

	return false
}

// filterWeekdays removes time ranges that do not have a Start time in the provided list of weekdays
func (trs *timeRanges) filterWeekdays(weekdays []time.Weekday) {
	if len(weekdays) <= 0 {
		return
	}

	trsNew := (*trs)[:0]
	for _, tr := range *trs {
		wdFound := false
		for _, wd := range weekdays {
			if tr.start.Weekday() == wd {
				wdFound = true
				break
			}
		}
		if wdFound {
			trsNew = append(trsNew, tr)
		}
	}

	*trs = trsNew
}

// FilterDates removes time ranges that do not have the same date as the provided list of times
func (trs *timeRanges) FilterDates(dates []time.Time) {
	if len(dates) <= 0 {
		return
	}

	trsNew := (*trs)[:0]
	for _, tr := range *trs {
		dateFound := false
		for _, date := range dates {
			if date.Year() == tr.start.Year() &&
				date.Month() == tr.start.Month() &&
				date.Day() == tr.start.Day() {
				dateFound = true
				break
			}
		}
		if dateFound {
			trsNew = append(trsNew, tr)
		}
	}

	*trs = trsNew
}
