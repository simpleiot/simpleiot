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

var usbDevices = []string{
	"/dev/sda1",
	"/dev/sda",
	"/dev/sdb1",
	"/dev/sdb",
	"/dev/sdc1",
	"/dev/sdc",
	"/dev/sdd1",
	"/dev/sdd",
}

// Check if the device is actually there at /dev/
func usbDeviceExists() bool {
	if runtime.GOARCH == "arm" {
		for _, d := range usbDevices {
			if file.Exists(d) {
				return true
			}
		}
	} else { // doesn't check for non-target systems
		return true
	}
	return false
}

func timeToMs(t time.Time) int64 {
	return t.UnixNano() / (1000 * 1000)
}

func timeToUs(t time.Time) int64 {
	return t.UnixNano() / (1000)
}

var tsFilenameFormat = "2006-01-02_15h04m05s"

// Run goroutine for data logging code
func Run(in, out chan interface{}, stateIn isdata.State, configIn isdata.Config, db *isdb.IsDb) {
	state := stateIn
	config := configIn

	var lastPulseTimestamp int64
	_ = lastPulseTimestamp

	var amount float64
	var amountTime time.Time

	logPulse := NewLog("is-"+state.SerialNumber+"-pulse", "timestamp(us),diff")
	logFlow := NewLog("is-"+state.SerialNumber+"-flow", "timestamp(us),amount,rate (GPH),average rate,pulses,shortWin")
	logPressure := NewLog("is-"+state.SerialNumber+"-pressure", "timestamp(us),average PSI,min,max")

	historyLogPeriod := 10 * time.Minute

	flowHistoryAvg := data.NewTimeWindowAverager(historyLogPeriod, func(avg data.Sample) {
		db.WriteSample(avg)
	}, isdata.SampleTypeFlowWindowAvg)

	presHistoryAvg := data.NewTimeWindowAverager(historyLogPeriod, func(avg data.Sample) {
		db.WriteSample(avg)
	}, isdata.SampleTypePressure)

	writeAmountTicker := time.NewTicker(historyLogPeriod)

	var alreadyExporting bool

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

			case isdata.ExportConfig:
				if alreadyExporting {
					out <- isdata.ExportAlreadyInProcess{}
					continue
				}
				go exportConfig(&config, &state, db, in)
				alreadyExporting = true

			case isdata.ExportData:
				// Start a goroutine for this large task so that
				// the log channel isn't overloaded
				// The function returns a signal when it is
				// finished: ExportDataFinished{}
				// The same thing happens for the next case
				if alreadyExporting {
					out <- isdata.ExportAlreadyInProcess{}
					continue
				}
				go exportHistoryData(&state, db, in)
				alreadyExporting = true

			case isdata.ExportFieldProductTotals:
				if alreadyExporting {
					out <- isdata.ExportAlreadyInProcess{}
					continue
				}
				go exportFieldTotals(&state, &config, in)
				alreadyExporting = true

			case isdata.ExportDataFinished,
				isdata.ExportConfigFinished,
				isdata.NoDiskPresent,
				isdata.ErrWriteDisk:
				// Update exporting status and send signal
				// to app channel so the dialog is updated
				alreadyExporting = false
				out <- m

			case isdata.Pulse:
				if !config.LogPulseData {
					continue
				}

				// Check for usb disk
				usbMountPoint := usbMountPoint()
				if usbMountPoint == "" || !usbDeviceExists() {
					out <- isdata.UpdateLogPulseEnable(false)
					out <- isdata.NoDiskPresent{}
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

				// Check for usb disk
				usbMountPoint := usbMountPoint()
				if usbMountPoint == "" || !usbDeviceExists() {
					out <- isdata.UpdateLogFlowEnable(false)
					out <- isdata.NoDiskPresent{}
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

					// Check for usb disk
					usbMountPoint := usbMountPoint()
					if usbMountPoint == "" || !usbDeviceExists() {
						out <- isdata.UpdateLogPressureEnable(false)
						out <- isdata.NoDiskPresent{}
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

				case isdata.SampleTypeInputInjector,
					isdata.SampleTypeInputIrrigator,
					isdata.SampleTypeInputWaterOn,
					isdata.SampleTypeMainAuxPwr,
					isdata.SampleTypeArm,
					isdata.SampleTypeFaultFlowOff,
					isdata.SampleTypeFaultPresLow,
					isdata.SampleTypeFaultPresHigh,
					isdata.SampleTypeFaultShutdown,
					isdata.SampleTypeFaultNtFlowOff,
					isdata.SampleTypeFaultNtPresLow,
					isdata.SampleTypeFaultNtPresHigh:
					err := db.WriteSample(m)
					if err != nil {
						log.Println("Error writing sample to database: ", err)
					}

				default:
					log.Println("Sample type not handled: ", m.Type)
				}

			default:
				log.Printf("Log Mux: unhandled message of type %T: %+v\r\n", m, m)
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
