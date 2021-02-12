// +build windows

package system

import "errors"

// EnableSyslog is not supported on windows
func EnableSyslog() error {
	return errors.New("Syslog not supported on windows")
}
