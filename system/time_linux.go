//go:build linux
// +build linux

package system

import (
	"log"
	"os/exec"
	"syscall"
	"time"
)

// SetTime sets the system time
func SetTime(t time.Time) (err error) {
	tv := syscall.NsecToTimeval(t.UnixNano())
	err = syscall.Settimeofday(&tv)
	if err != nil {
		log.Println("Error synchronizing system clock: ", err)
		return err
	}

	// Sync the real-time clock (RTC)
	// Always store time in UTC on the RTC
	err = exec.Command("hwclock", "-w", "-u").Run()
	if err != nil {
		return err
	}

	return nil
}
