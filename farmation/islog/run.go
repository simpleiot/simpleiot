package islog

// in logging, we write all timestamps as MS

import (
	"log"
	"os"
	"runtime"
	"sort"
	"strconv"
	"time"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/farmation/isdb"
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

// Run goroutine for ui code
func Run(in, out chan interface{}, db *isdb.IsDb) {
	config := isdata.Config{}
	var lastPulseTimestamp int64
	_ = lastPulseTimestamp

	logPulse := NewLog("pulse", "timestamp(us),diff")
	logFlow := NewLog("flow", "timestamp(us),rate (GPH)")
	//logAmount := NewLog("amount", "timestamp(us),amount")
	logPressure := NewLog("pressure", "timestamp(us),pressure (PSI),min,max,avg")
	logFault := NewLog("faults", "timestamp,fault")

	flowHistoryAvg := data.NewTimeWindowAverager(10*time.Minute, func(avg data.Sample) {
		db.WriteSample(avg)
	})
	presHistoryAvg := data.NewTimeWindowAverager(10*time.Minute, func(avg data.Sample) {
		db.WriteSample(avg)
	})
	amountHistoryAvg := data.NewTimeWindowAverager(10*time.Minute, func(avg data.Sample) {
		db.WriteSample(avg)
	})

	for {
		select {
		case m := <-in:
			switch m := m.(type) {
			case isdata.Config:
				config = m
				if !config.LogPulseData {
					lastPulseTimestamp = 0
					logPulse.Close()
				}
				if !config.LogFlowData {
					logFlow.Close()
				}
				if !config.LogPressureData {
					logPressure.Close()
				}

			case isdata.ExportData:
				// Extract faults from database
				faults, _ := db.ReadFaultHist()

				// Sort the faults by timestamp
				sort.Sort(faults)

				for _, fault := range faults {
					s := fault.Time.Format("2006-01-02T15:04:05Z07:00") + "," +
						fault.Fault.String()
					err := logFault.Write(s)
					if err != nil {
						log.Println("Error writing fault to file: ", err)
					}
				}

				err := file.SyncDisks()
				if err != nil {
					log.Println("sync error: ", err)
				}

				logFault.Close()

				// Extract samples from database
				samples, _ := db.ReadSamples()

				// Divide samples into flow, pressure, and amount samples
				var flows, pressures, amounts []data.Sample
				for _, sample := range samples {
					switch sample.Type {
					case isdata.SampleTypeFlowWindowAvg:
						flows = append(flows, sample)
					case isdata.SampleTypePressure:
						pressures = append(pressures, sample)
					case isdata.SampleTypeAmount:
						amounts = append(amounts, sample)
					}
				}

				// Sort the samples by timestamp
				/*sort.Sort(flows)
				sort.Sort(pressures)
				sort.Sort(amounts)*/

				for _, sample := range samples {
					s := sample.Time.Format("2006-01-02T15:04:05Z07:00") + "," +
						sample.Type
					err := logFault.Write(s)
					if err != nil {
						log.Println("Error writing sample to file: ", err)
					}
				}

				err = file.SyncDisks()
				if err != nil {
					log.Println("sync error: ", err)
				}

				logFault.Close()

				out <- isdata.ExportDataFinished{}

			case data.Sample:
				switch m.Type {
				case isdata.SampleTypePulses:
					if !config.LogPulseData {
						continue
					}

					tsMs := timeToUs(time.Time(m.Time))
					diff := tsMs - lastPulseTimestamp
					if lastPulseTimestamp == 0 {
						diff = 0
					}
					s := strconv.FormatInt(tsMs, 10) + "," + strconv.FormatInt(diff, 10)
					err := logPulse.Write(s)
					if err != nil {
						log.Println("Error writing pulse to file: ", err)
						out <- isdata.UpdateLogPulseEnable(false)
					}
					lastPulseTimestamp = tsMs

				case isdata.SampleTypeFlowInstantaneous:
					/*if !config.LogFlowData {
						continue
					}

					tsUs := timeToUs(m.Time)
					s := strconv.FormatInt(tsUs, 10) + "," +
						strconv.FormatFloat(m.Amount, 'f', 4, 64) + "," +
						strconv.FormatFloat(m.Rate, 'f', 1, 64) + "," +
						strconv.FormatFloat(m.RateAvg, 'f', 1, 64) + "," +
						strconv.Itoa(m.Pulses)
					err := logFlow.Write(s)
					if err != nil {
						log.Println("Error writing flow to file: ", err)
						out <- isdata.UpdateLogFlowEnable(false)
					}*/

				case isdata.SampleTypeFlowWindowAvg:
					// run flow sample through averager, which stores to
					// database every 10 minutes
					flowHistoryAvg.NewSample(m)

				case isdata.SampleTypePressure:
					// run pressure sample through averager, which stores to
					// database every 10 minutes
					presHistoryAvg.NewSample(m)

					// log data for engineering purpuses if enabled
					if !config.LogPressureData {
						continue
					}

					tsUs := timeToUs(m.Time)
					s := strconv.FormatInt(tsUs, 10) + "," +
						strconv.FormatFloat(m.Value, 'f', 2, 64) + "," +
						strconv.FormatFloat(m.Attributes["min"], 'f', 2, 64) + "," +
						strconv.FormatFloat(m.Attributes["max"], 'f', 2, 64) + "," +
						strconv.FormatFloat(m.Attributes["avg"], 'f', 2, 64)
					err := logPressure.Write(s)
					if err != nil {
						log.Println("Error writing pressure to file: ", err)
						out <- isdata.UpdateLogPressureEnable(false)
					}

				case isdata.SampleTypeAmount:
					// run amount sample through averager, which stores to
					// database every 10 minutes
					amountHistoryAvg.NewSample(m)
				}
			}
		}
	}
}
