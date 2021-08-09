package db

import (
	"fmt"
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

	if len(s.weekdays) > 0 {
		foundWeekday := false
		weekday := t.Weekday()
		for _, wd := range s.weekdays {
			if weekday == wd {
				foundWeekday = true
				break
			}
		}

		if !foundWeekday {
			return false, nil
		}
	}

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

	if !end.After(start) {
		fmt.Println("CLIFF: tUTC: ", tUTC)
		fmt.Println("CLIFF: start: ", start)
		fmt.Println("CLIFF: end: ", end)
		if tUTC.Before(start) && tUTC.After(end) {
			return false, nil
		}
	} else {
		// check if in time range
		if tUTC.Before(start) {
			return false, nil
		}

		if !tUTC.Before(end) {
			return false, nil
		}
	}

	return true, nil
}

var reHourMin = regexp.MustCompile(`(\d{1,2}):(\d\d)`)
var reDate = regexp.MustCompile(`(\d{4})-(\d{1,2})-(\d{1,2})`)

type timeRange struct {
	start time.Time
	end   time.Time
}
