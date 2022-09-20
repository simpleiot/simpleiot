package client

import (
	"testing"
	"time"
)

type testTime struct {
	t        time.Time
	expected bool
}

type testTable []testTime

func (tt *testTable) run(t *testing.T, sched *schedule) {
	for _, test := range *tt {
		active, err := sched.activeForTime(test.t)

		if err != nil {
			t.Errorf("got err: %v for time %v", err, test.t)
		}

		if active != test.expected {
			t.Errorf("expected %v for time %v", test.expected, test.t)
		}
	}
}

func TestScheduleAllDays(t *testing.T) {
	sched := newSchedule("2:00", "5:00", []time.Weekday{})

	tests := testTable{
		{time.Date(2021, time.February, 10, 4, 0, 0, 0, time.UTC), true},
		{time.Date(2021, time.February, 10, 5, 0, 0, 0, time.UTC), false},
	}

	tests.run(t, sched)
}

func TestScheduleWeekdays(t *testing.T) {
	sched := newSchedule("2:00", "5:00", []time.Weekday{0, 6})

	// 2021-08-09 is a Monday
	tests := testTable{
		{time.Date(2021, time.August, 8, 4, 0, 0, 0, time.UTC), true},
		{time.Date(2021, time.August, 10, 4, 0, 0, 0, time.UTC), false},
	}

	tests.run(t, sched)
}

func TestScheduleWrapDay(t *testing.T) {
	sched := newSchedule("20:00", "2:00", []time.Weekday{})

	// 2021-08-09 is a Monday
	tests := testTable{
		{time.Date(2021, time.August, 9, 21, 0, 0, 0, time.UTC), true},
		{time.Date(2021, time.August, 9, 1, 0, 0, 0, time.UTC), true},
	}

	tests.run(t, sched)
}

func TestScheduleWrapDayWeekday(t *testing.T) {
	sched := newSchedule("20:00", "2:00", []time.Weekday{1})

	// 2021-08-09 is a Monday
	tests := testTable{
		{time.Date(2021, time.August, 9, 21, 0, 0, 0, time.UTC), true},
		// the following should is not true as sched starts on previous
		// weekday
		{time.Date(2021, time.August, 9, 1, 0, 0, 0, time.UTC), false},
		{time.Date(2021, time.August, 10, 1, 0, 0, 0, time.UTC), true},
	}

	tests.run(t, sched)
}
