package islog

// in logging, we write all timestamps as MS

import (
	"log"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/file"
)

// usb disk is automounted by udev-extraconf

var usbMountPoints = []string{
	"/run/media/sda1",
	"/run/media/sda",
	"/run/media/sdb1",
	"/run/media/sdb",
	"/run/media/sdc1",
	"/run/media/sdc",
	"/run/media/sdd1",
	"/run/media/sdd",
}

func usbMountPoint() string {
	for _, d := range usbMountPoints {
		if file.Exists(d) {
			return d
		}
	}
	return ""
}

func timeToMs(t time.Time) int64 {
	return t.UnixNano() / (1000 * 1000)
}

// Run goroutine for ui code
func Run(in, out chan interface{}) {
	config := isdata.Config{}
	var pulseFile *os.File
	var lastPulseTimestamp int64

	for {
		select {
		case m := <-in:
			switch m := m.(type) {
			case isdata.Config:
				config = m
			case isdata.Pulse:
				if config.LogPulseData {
					if pulseFile == nil {
						usbMountPoint := usbMountPoint()
						if usbMountPoint != "" {
							fileName := "pulse-" + time.Now().Format(time.RFC3339) + ".csv"
							fileName = path.Join(usbMountPoint, fileName)
							var err error
							pulseFile, err = os.Create(fileName)
							if err != nil {
								log.Println("Error creating pulse file")
								pulseFile = nil
							}
						}
					}

					if pulseFile != nil {
						tsMs := timeToMs(time.Time(m))
						diff := tsMs - lastPulseTimestamp
						if lastPulseTimestamp == 0 {
							diff = 0
						}
						s := strconv.FormatInt(tsMs, 10) + "," + strconv.FormatInt(diff, 10) + "\n"
						_, err := pulseFile.Write([]byte(s))
						if err != nil {
							log.Println("Error writing pulse to file: ", err)
						}
						lastPulseTimestamp = tsMs
					}
				}
			}
		}
	}
}
