// +build darwin

package system

import (
	"errors"
	"time"
)

// SetTime sets the system time
func SetTime(t time.Time) (err error) {
	return errors.New("not implemented")
}
