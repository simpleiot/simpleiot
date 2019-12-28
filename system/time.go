package system

import (
	"log"
	"os/exec"
	"time"
)

// SetTime sets the system time to the
// parameter t with the date command
func SetTime(t time.Time) (err error) {

	tStr := t.Format("2006-01-02 15:04:05")

	err = exec.Command("date", "-s", tStr).Run()
	if err != nil {
		log.Println("Error setting system time: ", err)
		return err
	}

	return nil
}
