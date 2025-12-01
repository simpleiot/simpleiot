//go:build linux

package system

import (
	"log"
	"log/syslog"
)

// EnableSyslog enables logging to syslog
func EnableSyslog() error {
	lgr, err := syslog.New(syslog.LOG_NOTICE, "SIOT")
	if err != nil {
		return err
	}

	log.SetOutput(lgr)

	return nil
}
