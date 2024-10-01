package system

import (
	"log"

	"github.com/beevik/ntp"
)

// UpdateTimeFromNetwork fetches time from ntp server and stores in system and RTC
func UpdateTimeFromNetwork() (err error) {

	current, err := ntp.Time("0.pool.ntp.org")
	if err != nil {
		log.Println("Error fetching time from ntp.org:", err)
		return err
	}

	return SetTime(current)
}
