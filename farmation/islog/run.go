package islog

// in logging, we write all timestamps as MS

import (
	"log"
	"os"
	"runtime"
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

var tsFilenameFormat = "2006-01-02_15h04m05s"

// Run goroutine for data logging code
func Run(in, out chan interface{}, stateIn isdata.State, db *isdb.IsDb) {
	config := isdata.Config{}
	state := stateIn

	var lastPulseTimestamp int64
	_ = lastPulseTimestamp

	var amount float64
	var amountTime time.Time

	logPulse := NewLog("pulse", "timestamp(us),diff")
	logFlow := NewLog("flow", "timestamp(us),amount,rate (GPH),average rate,pulses,shortWin")
	logPressure := NewLog("pressure", "timestamp(us),average PSI,min,max")

	historyLogPeriod := 10 * time.Minute

	flowHistoryAvg := data.NewTimeWindowAverager(historyLogPeriod, func(avg data.Sample) {
		db.WriteSample(avg)
	}, isdata.SampleTypeFlowWindowAvg)

	presHistoryAvg := data.NewTimeWindowAverager(historyLogPeriod, func(avg data.Sample) {
		db.WriteSample(avg)
	}, isdata.SampleTypePressure)

	writeAmountTicker := time.NewTicker(historyLogPeriod)

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

			case isdata.State:
				state = m

			case isdata.ExportData:
				exportHistoryData(db, out)

			case isdata.ExportFieldProductTotals:
				exportFieldTotals(&state, &config, out)

			case isdata.Pulse:
				if !config.LogPulseData {
					continue
				}

				tsMs := timeToUs(time.Time(m))
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

			case isdata.Flow:
				if !config.LogFlowData {
					continue
				}

				tsUs := timeToUs(m.Time)
				s := strconv.FormatInt(tsUs, 10) + "," +
					strconv.FormatFloat(m.Amount, 'f', 4, 64) + "," +
					strconv.FormatFloat(m.Rate, 'f', 1, 64) + "," +
					strconv.FormatFloat(m.RateAvg, 'f', 1, 64) + "," +
					strconv.Itoa(m.Pulses) + "," +
					boolToYesNo(m.ShortWin)
				err := logFlow.Write(s)
				if err != nil {
					log.Println("Error writing flow to file: ", err)
					out <- isdata.UpdateLogFlowEnable(false)
				}

			case data.Sample:
				switch m.Type {
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
						strconv.FormatFloat(m.Min, 'f', 2, 64) + "," +
						strconv.FormatFloat(m.Max, 'f', 2, 64)
					err := logPressure.Write(s)
					if err != nil {
						log.Println("Error writing pressure to file: ", err)
						out <- isdata.UpdateLogPressureEnable(false)
					}

				case isdata.SampleTypeAmount:
					// accumulate amount over 10m
					amount += m.Value
					if m.Time.After(amountTime) {
						amountTime = m.Time
					}
				}
			}

		case <-writeAmountTicker.C:
			db.WriteSample(data.Sample{
				Type:  isdata.SampleTypeAmount,
				Time:  amountTime,
				Value: amount,
			})

			// reset amount and time
			amount = 0
			amountTime = time.Now()
		}
	}
}

func boolToYesNo(val bool) string {
	if val {
		return "yes"
	}
	return "no"
}
