package app

import (
	"errors"
	"log"
	"runtime"

	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/system"
)

// InitSystemTimezone sets the system timezone from the config setting
func InitSystemTimezone(params Params) error {
	if runtime.GOARCH != "arm" {
		return errors.New("We only set timezone on target systems")
	}

	config := isdata.Config{}
	state := isdata.State{}

	_, _, _, err := DbInit(params.DataDir, &config, &state)

	if err != nil {
		return err
	}

	// Check that the system timezone didn't get messed up
	zonePath, zone, err := system.GetTimezone()
	if err != nil {
		log.Println("Error fetching current timezone: ", err)
	}

	if zone != config.Timezone || zonePath != "US" {
		log.Println("setting timezone to: ", config.Timezone)

		err = system.SetTimezone("US", config.Timezone)
		if err != nil {
			log.Println("Error setting timezone: ", err)
		}
	}

	return nil
}
