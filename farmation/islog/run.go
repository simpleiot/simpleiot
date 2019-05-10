package islog

// in logging, we write all timestamps as MS

import (
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
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
	if runtime.GOARCH == "arm" {
		for _, d := range usbMountPoints {
			if file.Exists(d) {
				return d
			}
		}
	} else {
		dir, err := os.Getwd()
		if err == nil {
			return dir
		}
	}
	return ""
}

func timeToMs(t time.Time) int64 {
	return t.UnixNano() / (1000 * 1000)
}

func timeToUs(t time.Time) int64 {
	return t.UnixNano() / (1000)
}

var tsFilenameFormat = "2006-01-02T150405Z07:00"

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

// Run goroutine for ui code
func Run(in, out chan interface{}) {
	config := isdata.Config{}
	var pulseFile *os.File
	var flowFile *os.File
	var lastPulseTimestamp int64

	stopPulseLog := func() {
		config.LogPulseData = false
		if pulseFile != nil {
			log.Println("Closing pulse log file")
			pulseFile.Close()
		}
		pulseFile = nil
		lastPulseTimestamp = 0
	}

	stopFlowLog := func() {
		config.LogFlowData = false
		if flowFile != nil {
			log.Println("Closing flow log file")
			flowFile.Close()
		}
		flowFile = nil
	}

	for {
		select {
		case m := <-in:
			switch m := m.(type) {
			case isdata.Config:
				config = m
				if !config.LogPulseData {
					stopPulseLog()
				}
				if !config.LogFlowData {
					stopFlowLog()
				}
			case isdata.Flow:
				if config.LogFlowData {
					if flowFile == nil {
						usbMountPoint := usbMountPoint()
						if usbMountPoint == "" {
							// disable flow logging as there is no where to send the data
							out <- isdata.UpdateLogFlowEnable(false)
						} else {
							var err error
							flowFile, err = createLogFile("flow")
							if err != nil {
								fmt.Println("Error creating flow log file: ", err)
								out <- isdata.UpdateLogFlowEnable(false)
							} else {
								flowFile.Write([]byte("timestamp(us),amount,rate (GPH),average rate,pulses\n"))
							}
						}
					}

					if flowFile != nil {
						tsUs := timeToUs(m.Time)
						s := strconv.FormatInt(tsUs, 10) + "," +
							strconv.FormatFloat(m.Amount, 'f', 4, 64) + "," +
							strconv.FormatFloat(m.Rate, 'f', 1, 64) + "," +
							strconv.FormatFloat(m.RateAvg, 'f', 1, 64) + "," +
							strconv.Itoa(m.Pulses) + "\n"
						_, err := flowFile.Write([]byte(s))
						if err != nil {
							log.Println("Error writing flow to file: ", err)
							stopFlowLog()
						}
					}
				}
			case isdata.Pulse:
				if config.LogPulseData {
					if pulseFile == nil {
						usbMountPoint := usbMountPoint()
						if usbMountPoint == "" {
							// disable pulse logging as there is no where to send the data
							out <- isdata.UpdateLogPulseEnable(false)
						} else {
							var err error
							pulseFile, err = createLogFile("pulse")
							if err != nil {
								fmt.Println("Error creating pulse log file: ", err)
								out <- isdata.UpdateLogPulseEnable(false)
							} else {
								pulseFile.Write([]byte("timestamp(us),diff\n"))
							}
						}
					}

					if pulseFile != nil {
						tsMs := timeToUs(time.Time(m))
						diff := tsMs - lastPulseTimestamp
						if lastPulseTimestamp == 0 {
							diff = 0
						}
						s := strconv.FormatInt(tsMs, 10) + "," + strconv.FormatInt(diff, 10) + "\n"
						_, err := pulseFile.Write([]byte(s))
						if err != nil {
							log.Println("Error writing pulse to file: ", err)
							stopPulseLog()
						}
						lastPulseTimestamp = tsMs
					}
				}
			}
		}
	}
}
