package system

import (
	"os/exec"
	"time"
)

// SetTime sets the system time to the
// parameter t with the date command
func SetTime(t time.Time) (err error) {

	tStr := t.Format("2006-01-02 15:04:05")

	err = exec.Command("date", "-s", tStr).Run()
	if err != nil {
		return err
	}

	// Sync the real-time clock
	err = exec.Command("hwclock", "-w").Run()
	if err != nil {
		return err
	}

	return nil
}
