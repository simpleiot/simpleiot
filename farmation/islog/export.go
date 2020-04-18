package islog

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/farmation/isdb"
	"github.com/simpleiot/simpleiot/file"
)

func exportConfig(config *isdata.Config, state *isdata.State, db *isdb.IsDb, out chan interface{}) {

	// check if disk present before reading from database,
	// because read takes time
	usbMountPoint := usbMountPoint()
	if usbMountPoint == "" || !usbDeviceExists() {
		out <- isdata.NoDiskPresent{}
		return
	}
	fn := "is-" + state.SerialNumber + "_" + time.Now().Format(tsFilenameFormat) + ".config"

	fn = path.Join(usbMountPoint, fn)

	f, err := os.Create(fn)

	if err != nil {
		log.Println("Error opening config file: ", err)
		out <- isdata.ErrWriteDisk{}
		return
	}

	// Sync disks
	defer func() {
		f.Close()
		err := file.SyncDisks()
		if err != nil {
			log.Println("Error syncing disks: ", err)
		}
	}()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "   ")

	err = encoder.Encode(config)

	if err != nil {
		log.Println("Error encoding config")
		out <- isdata.ErrWriteDisk{}
		return
	}

	// Send out finished signal and return
	out <- isdata.ExportConfigFinished{}
	return
}

func exportHistoryData(state *isdata.State, db *isdb.IsDb, out chan interface{}) {

	// check if disk present before reading from database,
	// because read takes time
	usbMountPoint := usbMountPoint()
	if usbMountPoint == "" || !usbDeviceExists() {
		out <- isdata.NoDiskPresent{}
		return
	}

	historyData := NewLog("is-"+state.SerialNumber+"-data", "timestamp (us),type,value,min,max")

	// Sync disks
	defer func() {
		historyData.Close()
		err := file.SyncDisks()
		if err != nil {
			log.Println("Error syncing disks: ", err)
		}
	}()

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

		case isdata.SampleTypeFaultFlowOff,
			isdata.SampleTypeFaultPresLow,
			isdata.SampleTypeFaultPresHigh,
			isdata.SampleTypeFaultNtFlowOff,
			isdata.SampleTypeFaultNtPresLow,
			isdata.SampleTypeFaultNtPresHigh:
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

	// Send out finished signal and return
	out <- isdata.ExportDataFinished{}
	return
}

func exportFieldTotals(state *isdata.State, config *isdata.Config, out chan interface{}) {

	// Check for usb disk
	usbMountPoint := usbMountPoint()
	if usbMountPoint == "" || !usbDeviceExists() {
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
	data = append(data, []string{"DEVICE", config.DeviceName},
		[]string{"SERIAL #", state.SerialNumber}, []string{"DATE", tStamp}, []string{})

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
		out <- isdata.ErrWriteDisk{}
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
	err = totals.MergeCell("Sheet1", "B1", "F1")
	if err != nil {
		log.Println("Error merging cells: ", err)
	}
	err = totals.MergeCell("Sheet1", "B2", "F2")
	if err != nil {
		log.Println("Error merging cells: ", err)
	}
	err = totals.MergeCell("Sheet1", "B3", "F3")
	if err != nil {
		log.Println("Error merging cells: ", err)
	}
	err = totals.MergeCell("Sheet1", "A4", "F4")
	if err != nil {
		log.Println("Error merging cells: ", err)
	}

	// Save file
	fileName := "is-" + state.SerialNumber + "-totals_" + tStamp + ".xlsx"

	// If runtime.GOARCH != "arm", usbMountPoint will be either the working
	// directory or an empty string, either of which works for method SaveAs.
	fileName = path.Join(usbMountPoint, fileName)

	fmt.Println("Saving", fileName)

	err = totals.SaveAs(fileName)
	if err != nil {
		log.Println("Error saving "+fileName+": ", err)
	}

	// Send out finished signal and return
	out <- isdata.ExportDataFinished{}
	return
}

func exportSystemLogs(state *isdata.State, out chan interface{}) {

	usbMountPoint := usbMountPoint()
	if usbMountPoint == "" || !usbDeviceExists() {
		out <- isdata.NoDiskPresent{}
		return
	}

	fn := "is-" + state.SerialNumber + "_syslogs_" + time.Now().Format(tsFilenameFormat) + ".zip"

	fn = path.Join(usbMountPoint, fn)

	zipFile, err := os.Create(fn)
	if err != nil {
		log.Println("Error creating logs zip file on USB: ", err)
		out <- isdata.ErrWriteDisk{}
		return
	}

	zipWriter := zip.NewWriter(zipFile)

	// Sync disks and close output file and writer
	defer func() {
		zipWriter.Close()
		zipFile.Close()
		err := file.SyncDisks()
		if err != nil {
			log.Println("Error syncing disks: ", err)
		}
	}()

	// Read file info for all of the message files in the log
	// directory into a slice
	msgFileDir := "/data/log/"
	msgFileInfos, err := ioutil.ReadDir(msgFileDir)
	if err != nil {
		log.Println("Error reading info for message files from log dir: ", err)
		return
	}

	for _, msgFileInfo := range msgFileInfos {

		msgFilePath := path.Join(msgFileDir, msgFileInfo.Name())

		// Open the current file using its name from
		// the file info
		msgFile, err := os.Open(msgFilePath)
		if err != nil {
			log.Println("Error opening a log message file: ", err)
			out <- isdata.ErrWriteDisk{}
			return
		}

		// The message files will be closed in reverse order due to the way defer
		// works, but this shouldn't matter
		defer msgFile.Close()

		// Create file header from the file info
		header, err := zip.FileInfoHeader(msgFileInfo)
		if err != nil {
			log.Println("Error creating log message file header: ", err)
		}
		// Specify full path
		header.Name = msgFilePath
		// Use compression
		header.Method = zip.Deflate

		// Create an io.Writer
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			log.Println("Error creating zip writer for logs: ", err)
		}

		// Write data to io.Writer type
		_, err = io.Copy(writer, msgFile)
		if err != nil {
			log.Println("Error writing log message file: ", err)
		}
	}

	// Send out finished signal and return
	out <- isdata.ExportDataFinished{}
	return
}

func boolToString(val bool) string {
	if val {
		return "on"
	}
	return "off"
}
