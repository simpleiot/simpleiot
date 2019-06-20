package islog

import (
	"io"
	"log/syslog"
)

// Syslog returns a new syslog writer
func Syslog() (io.Writer, error) {
	return syslog.New(syslog.LOG_NOTICE, "IS")
}
