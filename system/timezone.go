package system

import (
	"log"
	"os"
)

// GetTimezone returns the current system time zone
func GetTimezone() (string, error) {
	link, err := os.Readlink("/etc/localtime")
	if err != nil {
		log.Println("Error finding time zone, ", err)
		return "", err
	}

	return link, nil
}

// SetTimezone sets the current system time zone
// Takes *US* time zones
func SetTimezone(zone string) error {
	if _, err := os.Lstat("/etc/localtime"); err == nil {
		err := os.Remove("/etc/localtime")
		if err != nil {
			log.Println("Error removing old time zone link, ", err)
			return err
		}
	}

	err := os.Symlink("/usr/share/zoneinfo/US/"+zone, "/etc/localtime")
	if err != nil {
		log.Println("Error linking to new time zone, ", err)
		return err
	}

	return nil
}

// ReadTimezones returns a list of possible time zones
// from the system
// Returns *US* time zones
func ReadTimezones() (list []string, err error) {

	file, err := os.Open("/usr/share/zoneinfo/US/")
	if err != nil {
		log.Println("Error opening time zone file, ", err)
		return nil, err
	}

	fileInfo, err := file.Readdir(-1)
	if err != nil {
		log.Println("Error reading time zones, ", err)
		return nil, err
	}

	for _, fi := range fileInfo {
		list = append(list, fi.Name())
	}

	return list, nil
}
