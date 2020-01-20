package islog

import (
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/farmation/isdb"
	"github.com/simpleiot/simpleiot/file"
)

func exportHistoryData(state *isdata.State, db *isdb.IsDb, out chan interface{}) {

	// check if disk present before reading from database,
	// because read takes time
	usbMountPoint := usbMountPoint()
	if usbMountPoint == "" {
		out <- isdata.NoDiskPresent{}
		return
	}

	historyData := NewLog("is-"+state.SerialNumber+"-data", "timestamp (us),type,value,min,max")

	defer historyData.Close()

	// Sync disks
	defer func() {
		err := file.SyncDisks()
		if err != nil {
			log.Println("Error syncing disks: ", err)
		}
	}()

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

func exportFieldTotals(state *isdata.State, config *isdata.Config, out chan interface{}) {

	// Check for usb disk
	usbMountPoint := usbMountPoint()
	if usbMountPoint == "" {
		out <- isdata.NoDiskPresent{}
		return
	}

	// Sync disks when all other processes finish
	defer func() {
		err := file.SyncDisks()
		if err != nil {
			log.Println("Error syncing disks: ", err)
		}
	}()

	// Format data as a 2D array of strings
	var data [][]string

	// Add unit serial number, date, and blank row
	tStamp := time.Now().Format(tsFilenameFormat)
	data = append(data, []string{"SERIAL #", state.SerialNumber}, []string{"DATE", tStamp}, []string{})

	// Add product labels

	// Initialize with one empty string to get correct offset in spreadsheet
	productLabels := []string{""}

	for _, productConfig := range config.ProductConfigs {
		productLabels = append(productLabels, productConfig.Description)
	}
	data = append(data, productLabels)

	// Add all of the totals to the array
	var productStrTotals []string

	for i, productStates := range state.FieldStates {

		// Add the field name to the beginning of the array
		productStrTotals = append(productStrTotals, config.FieldConfigs[i].Description)

		// Add each total by product
		for _, productState := range productStates {
			productStrTotals = append(productStrTotals,
				strconv.FormatFloat(productState.Total, 'f', 2, 64))
		}

		data = append(data, productStrTotals)
		productStrTotals = nil
	}

	// Format and save array as a .xlsx file
	totals, err := ArrayToSpreadsheet(data)
	if err != nil {
		log.Println("Error converting to spreadsheet format: ", err)
		return
	}

	// Set product labels bold
	bold, err := totals.NewStyle(`{"font":{"bold":true}}`)
	if err != nil {
		log.Println("Error creating style: ", err)
	}
	err = totals.SetCellStyle("Sheet1", "B4", "F4", bold)
	if err != nil {
		log.Println("Error setting cell style: ", err)
	}

	// Set field labels italic
	italic, err := totals.NewStyle(`{"font":{"italic":true}}`)
	if err != nil {
		log.Println("Error creating style: ", err)
	}
	err = totals.SetCellStyle("Sheet1", "A5", "A35", italic)
	if err != nil {
		log.Println("Error setting cell style: ", err)
	}

	// Make cells large enough for any field/product name
	err = totals.SetColWidth("Sheet1", "A", "F", 11)
	if err != nil {
		log.Println("Error setting column width: ", err)
	}

	// Merge unused cells in top three lines
	err = totals.MergeCell("Sheet1", "C1", "F1")
	if err != nil {
		log.Println("Error merging cells: ", err)
	}
	err = totals.MergeCell("Sheet1", "B2", "D2")
	if err != nil {
		log.Println("Error merging cells: ", err)
	}
	err = totals.MergeCell("Sheet1", "E2", "F2")
	if err != nil {
		log.Println("Error merging cells: ", err)
	}
	err = totals.MergeCell("Sheet1", "A3", "F3")
	if err != nil {
		log.Println("Error merging cells: ", err)
	}

	// Save file
	err = totals.SaveAs("is-" + state.SerialNumber + "-totals_" + tStamp + ".xlsx")
	if err != nil {
		log.Println("Error saving .xlsx file: ", err)
	}

	// Send out finished signal - close dialog
	out <- isdata.ExportDataFinished{}
}

func boolToString(val bool) string {
	if val {
		return "on"
	}
	return "off"
}
