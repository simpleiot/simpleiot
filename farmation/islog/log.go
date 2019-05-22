package islog

import (
	"errors"
	"log"
	"os"
	"path"
	"time"
)

func createLogFile(basename string) (*os.File, error) {
	fileName := basename + "-" + time.Now().Format(tsFilenameFormat) + ".csv"
	fileName = path.Join(usbMountPoint(), fileName)
	var err error
	log.Println("Creating: ", fileName)
	retFile, err := os.Create(fileName)
	if err != nil {
		return nil, err
	}

	return retFile, nil
}

// Log represents a log file that can be written to USB flash disk
type Log struct {
	name    string
	heading string
	file    *os.File
}

// NewLog creates a new log file
func NewLog(name, heading string) *Log {
	return &Log{name: name, heading: heading}
}

// Close closes the log file
func (l *Log) Close() {
	if l.file != nil {
		log.Println("Closing " + l.name + " log file")
		l.file.Close()
		l.file = nil
	}
}

var errNoUsbDisk = errors.New("No USB disk present")

// Write writes a string to a log file
// returns error if USB disk is not present, etc
func (l *Log) Write(line string) error {
	if l.file == nil {
		usbMountPoint := usbMountPoint()
		if usbMountPoint == "" {
			return errNoUsbDisk
		}

		var err error
		l.file, err = createLogFile(l.name)
		if err != nil {
			log.Println("Error creating log file: ", err)
			return err
		}

		l.file.Write([]byte(l.heading + "\n"))
	}

	if l.file != nil {
		_, err := l.file.Write([]byte(line + "\n"))
		if err != nil {
			l.Close()
			return err
		}
	}

	return nil
}
