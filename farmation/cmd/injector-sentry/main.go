package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"net/http"
	//_ "net/http/pprof"

	ps "github.com/mitchellh/go-ps"
	"github.com/simpleiot/simpleiot/db"
	"github.com/simpleiot/simpleiot/farmation/app"
	"github.com/simpleiot/simpleiot/farmation/diag"
	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/farmation/isdb"
	"github.com/simpleiot/simpleiot/farmation/isio"
	"github.com/simpleiot/simpleiot/farmation/islog"
	"github.com/simpleiot/simpleiot/network"
)

func main() {
	flagSim := flag.Bool("sim", false, "Start device simulator")
	flagDiagRun := flag.Bool("diagRun", false, "Run device diagnostics")
	flagDiagList := flag.Bool("diagList", false, "List available diagnostics")
	flagDiagSingle := flag.String("diagSingle", "", "Run single test")
	flagDebugState := flag.Bool("debugState", false, "log state changes")
	flagDebugConfig := flag.Bool("debugConfig", false, "log config changes")
	flagDebugPortal := flag.Bool("debugPortal", false, "debug portal communication")
	flagDebugModem := flag.Bool("debugModem", false, "enable modem debugging")
	flagSyslog := flag.Bool("syslog", false, "log to syslog instead of stdout")
	flagDataDir := flag.String("datadir", "", "directory to store data in")
	flagReadPressure := flag.Bool("readPressure", false, "read pressure sensor")
	flagCheckDb := flag.String("checkDb", "", "check database")
	/*
		flagModemState := flag.Bool("modemState", false, "read modem state")
		flagModemSettings := flag.Bool("modemSettings", false, "read modem settings")
		flagModemInfo := flag.Bool("modemInfo", false, "read modem info")
		flagModemConfigure := flag.Bool("modemConfigure", false, "configure modem")
		flagModemGet := flag.Bool("modemGet", false, "execute modem get request")
	*/
	flagModemStatus := flag.Bool("modemStatus", false, "get modem status")
	flagModemReset := flag.Bool("modemReset", false, "reset modem")
	flagPortal := flag.String("portal", "https://portal.farmation.us", "portal URL")
	flagSerialNumber := flag.String("serialNumber", "", "IS serial number")
	flagSetIsSN := flag.String("setSN", "", "Set IS serial number")
	flagEnableAuxRelay := flag.Bool("enableAuxRelay", false, "enable aux relay")
	flagHwID := flag.Bool("hwId", false, "display HW ID")
	flagViewMsg := flag.Bool("msg", false, "view channel messages to app")
	flagReadVcap := flag.Bool("readVcap", false, "read backup battery voltage")
	flagWebUI := flag.Bool("webUI", false, "Start Web UI for remote access")
	flagProf := flag.Bool("prof", false, "Web UI for profiling")
	flagSetTimeZone := flag.Bool("setTimeZone", false, "Set system time zone from config")
	flagPopDbTestData := flag.Bool("popDbTestData", false, "Populate db with 10yr worth of data")
	flagDbDumpSamples := flag.Bool("dbDumpSamples", false, "Dump samples in DB")
	flagDbCountSamples := flag.Bool("dbCountSamples", false, "Count samples in DB")
	flagDbDumpFaults := flag.Bool("dbDumpFaults", false, "Dump faults")
	flag.Parse()

	if *flagDiagRun {
		isio.GpioInit()
		diag.Run()
		return
	}

	if *flagDiagList {
		diag.List()
		return
	}

	if *flagDiagSingle != "" {
		isio.GpioInit()
		diag.RunSingle(*flagDiagSingle)
		return
	}

	if *flagHwID {
		isio.GpioInit()
		fmt.Println("HW ID: ", isio.GetHwID())
		os.Exit(0)
	}

	if *flagReadPressure {
		isio.GpioInit()
		ref, sense, err := isio.ReadPressure()
		if err != nil {
			log.Println("Error reading pressure sensor: ", err)
			os.Exit(-1)
		}

		pres := isio.CalcPressure(ref, sense, 250)

		log.Printf("Pressure ref: %v, sense: %v, pres: %v\n", ref, sense, pres)
		os.Exit(0)
	}

	if *flagSyslog {
		logwriter, err := islog.Syslog()
		if err == nil {
			log.SetOutput(logwriter)
		} else {
			log.Println("Error sending log to syslog: ", err)
		}
	}

	if *flagCheckDb != "" {
		err := db.BBoltCheck(*flagCheckDb)
		if err != nil {
			log.Println("check failed: ", err)
			os.Exit(-1)
		}
		log.Println("check passed")
		os.Exit(0)
	}

	if *flagSetTimeZone {
		err := app.InitSystemTimezone(*flagDataDir)
		if err != nil {
			log.Println("Error initializing time zones: ", err)
			os.Exit(-1)
		}

		os.Exit(0)
	}

	if *flagPopDbTestData {
		err := isdb.PopDbTestData(*flagDataDir)
		if err != nil {
			log.Println("Error populating database data: ", err)
			os.Exit(-1)
		}

		os.Exit(0)
	}

	if *flagDbDumpSamples {
		err := isdb.DbDumpSamples(*flagDataDir)
		if err != nil {
			log.Println("Error populating database data: ", err)
			os.Exit(-1)
		}

		os.Exit(0)
	}

	if *flagDbCountSamples {
		config := isdata.Config{}
		state := isdata.State{}

		_, _, dbData, err := isdb.DbInit(*flagDataDir, &config, &state)

		if err != nil {
			log.Println("Error opening database: ", err)
			os.Exit(-1)
		}

		start := time.Now()
		count, err := dbData.GetSampleCount()
		if err != nil {
			log.Println("Error getting sample count: ", err)
			os.Exit(-1)
		}
		log.Println("Samples in DB: ", count)
		log.Println("Took: ", time.Since(start))

		os.Exit(0)
	}

	if *flagDbDumpFaults {
		config := isdata.Config{}
		state := isdata.State{}

		_, _, dbData, err := isdb.DbInit(*flagDataDir, &config, &state)

		if err != nil {
			log.Println("Error opening database: ", err)
			os.Exit(-1)
		}

		start := time.Now()
		faults, err := dbData.ReadFaultHist(start.AddDate(0, 0, -7))
		if err != nil {
			log.Println("Error getting sample count: ", err)
			os.Exit(-1)
		}
		for _, f := range faults {
			fmt.Printf("%+v\n", f)
		}
		log.Printf("Fault count: %v, Took: %v", len(faults), time.Since(start))

		os.Exit(0)
	}

	if *flagModemReset {
		isio.GpioInit()
		isio.ResetModem()
		log.Println("modem reset")
		os.Exit(0)
	}

	if *flagModemStatus {
		isio.GpioInit()
		modem := network.NewModem(
			network.ModemConfig{
				ChatScript:    "bg96",
				AtCmdPortName: "/dev/ttyUSB2",
				Reset:         isio.ResetModem,
				Debug:         false,
				APN:           "vzwinternet",
			})

		status, err := modem.GetStatus()
		if err != nil {
			log.Println("Error getting modem status: ", err)
			os.Exit(-1)
		}

		fmt.Printf("Modem status: %+v\n", status)
		os.Exit(0)
	}

	if *flagSetIsSN != "" {
		if runtime.GOARCH != "arm" {
			log.Println("Error: Can only set SN on IS")
			os.Exit(-1)
		}

		err := isdb.WriteSerialNumber(*flagSetIsSN)
		if err != nil {
			log.Println("Error writing SN: ", err)
			os.Exit(-1)
		}

		log.Println("Serial number set, please restart system")
		os.Exit(0)
	}

	if *flagEnableAuxRelay {
		isio.GpioInit()
		isio.GpioOut(isio.GpioRelayAuxEn, true)
		os.Exit(0)
	}

	// check if app is already running -- don't want to run two
	// copies of the app
	if runtime.GOARCH == "arm" {
		processes, err := ps.Processes()
		if err != nil {
			log.Println("Error getting processes")
		}

		count := 0

		for _, p := range processes {
			if p.Executable() == "is" || p.Executable() == "is_arm" {
				count++
				if count > 1 {
					log.Println("is app already running, bailing")
					os.Exit(-1)
				}
			}
		}
	}

	if *flagProf {
		// this starts a web service that can be used for profiling
		// must uncomment import _ "net/http/pprof" above
		go func() {
			log.Println("Starting web interface for pprof ...")
			log.Println(http.ListenAndServe(":6060", nil))
		}()
	}

	params := app.Params{
		Sim:          *flagSim,
		DataDir:      *flagDataDir,
		DebugState:   *flagDebugState,
		DebugConfig:  *flagDebugConfig,
		DebugModem:   *flagDebugModem,
		DebugPortal:  *flagDebugPortal,
		PortalURL:    *flagPortal,
		SerialNumber: *flagSerialNumber,
		ViewMsg:      *flagViewMsg,
		ReadVcap:     *flagReadVcap,
		WebUI:        *flagWebUI,
	}

	app.Run(params)
}
