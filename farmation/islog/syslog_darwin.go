package islog

import (
	"errors"
	"io"
)

// Syslog returns a new syslog writer
func Syslog() (io.Writer, error) {
	return nil, errors.New("syslog not supported on windows")
}
