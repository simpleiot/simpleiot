package isui

import (
	"strconv"
	"time"
)

// Date formats a time into strings:
// yyyy, mm, dd if fullYear is true and yy, mm, dd otherwise
func Date(t time.Time, fullYear bool) (string, string, string) {
	y, m, d := t.Date()
	yS, mS, dS := strconv.Itoa(int(y)), strconv.Itoa(int(m)), strconv.Itoa(int(d))

	if !fullYear {
		// if yyyy, make yy
		if len(yS) > 2 {
			yS = yS[2:]
		}
	}

	return yS, mS, dS
}

// Clock formats a time into a clock time: hh, mm, and ss
func Clock(t time.Time) (string, string, string) {
	h, m, s := t.Clock()
	hS, mS, sS := AddZero(strconv.Itoa(h)), AddZero(strconv.Itoa(m)), AddZero(strconv.Itoa(s))
	return AddZero(hS), AddZero(mS), AddZero(sS)
}

// AddZero adds a 0 in front of one digit values
func AddZero(a string) string {
	if len(a) <= 1 {
		a = "0" + a
	}
	return a
}
