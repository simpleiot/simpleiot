package system

import (
	"log"
	"os"
)

// ReadTimezones returns a list of possible time zones
// from the system
// Possible arguments for zoneInfoDir:
//	"" (root dir)
//	"US"
//	"posix/America"
func ReadTimezones(zoneInfoDir string) (list []string, err error) {

	zoneInfoPath := setZoneInfoPath(zoneInfoDir)

	file, err := os.Open(zoneInfoPath)
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
		if !fi.IsDir() { // if file, not directory
			list = append(list, fi.Name())
		}
	}

	return list, nil
}

// GetTimezone returns the current system time zone
func GetTimezone() (string, error) {

	link, err := os.Readlink(zoneLink)
	if err != nil {
		log.Println("Error finding time zone, ", err)
		return "", err
	}

	return link, nil
}

// SetTimezone sets the current system time zone
func SetTimezone(zoneInfoDir, zone string) error {

	if _, err := os.Lstat(zoneLink); err == nil {
		err := os.Remove(zoneLink)
		if err != nil {
			log.Println("Error removing old time zone link, ", err)
			return err
		}
	}

	zoneInfoPath := setZoneInfoPath(zoneInfoDir)

	err := os.Symlink(zoneInfoPath+zone, zoneLink)
	if err != nil {
		log.Println("Error linking to new time zone, ", err)
		return err
	}

	return nil
}

// Symbolic link for the system timezone
const zoneLink = "/etc/localtime"

func setZoneInfoPath(zoneInfoDir string) (zoneInfoPath string) {
	zoneInfoPath = "/usr/share/zoneinfo/"
	if zoneInfoDir == "" {
		return zoneInfoPath
	}
	return zoneInfoPath + zoneInfoDir + "/"
}
