package islog

import (
	"errors"
	"log"
	"strconv"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/farmation/isdb"
	"github.com/simpleiot/simpleiot/file"
)

func exportHistoryData(db *isdb.IsDb, out chan interface{}) {

	// check if disk present before reading from database,
	// because read takes time
	usbMountPoint := usbMountPoint()
	if usbMountPoint == "" {
		out <- isdata.NoDiskPresent{}
		return
	}

	historyData := NewLog("is-data", "timestamp (us),type,value,min,max")

	defer historyData.Close()
	defer file.SyncDisks()

	var errNoUsbDisk = errors.New("No USB disk present")

	// Extract samples from database
	samples, _ := db.ReadSamples()

	// Write samples to disk
	for _, sample := range samples {
		var s string
		switch sample.Type {
		case isdata.SampleTypeFlowWindowAvg, isdata.SampleTypePressure:
			s = sample.Time.Format("2006-01-02T15:04:05Z07:00") + "," +
				sample.Type + "," +
				strconv.FormatFloat(sample.Value, 'f', 2, 64) + "," +
				strconv.FormatFloat(sample.Min, 'f', 2, 64) + "," +
				strconv.FormatFloat(sample.Max, 'f', 2, 64)

		case isdata.SampleTypeAmount:
			s = sample.Time.Format("2006-01-02T15:04:05Z07:00") + "," +
				sample.Type + "," +
				strconv.FormatFloat(sample.Value, 'f', 2, 64) + "," +
				"-," +
				"-"

		case isdata.SampleTypeInputInjector, isdata.SampleTypeInputIrrigator, isdata.SampleTypeInputWaterOn, isdata.SampleTypeArm:
			s = sample.Time.Format("2006-01-02T15:04:05Z07:00") + "," +
				sample.Type + "," +
				boolToString(sample.Bool()) + "," +
				"-," +
				"-"

		case isdata.SampleTypeFaultFlowOff, isdata.SampleTypeFaultPresLow:
			s = sample.Time.Format("2006-01-02T15:04:05Z07:00") + "," +
				sample.Type + "," +
				strconv.FormatFloat(sample.Value, 'f', 2, 64) + "," +
				"-," +
				"-"

		case isdata.SampleTypeFaultShutdown, data.SampleTypeStartApp:
			s = sample.Time.Format("2006-01-02T15:04:05Z07:00") + "," +
				sample.Type + "," +
				"-," +
				"-," +
				"-"

		default:
			log.Println("Log: unhandled sample: ", sample)
			continue
		}

		err := historyData.Write(s)
		if err != nil {
			log.Println("Error writing sample to file: ", err)
			if err == errNoUsbDisk {
				out <- isdata.NoDiskPresent{}
			} else {
				out <- isdata.ErrWriteDisk{}
			}
			return
		}
	}

	out <- isdata.ExportDataFinished{}
}

func boolToString(val bool) string {
	if val {
		return "on"
	}
	return "off"
}
