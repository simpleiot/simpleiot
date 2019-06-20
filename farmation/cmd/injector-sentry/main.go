package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/simpleiot/simpleiot/farmation/app"
	"github.com/simpleiot/simpleiot/farmation/diag"
	"github.com/simpleiot/simpleiot/farmation/isio"
	"github.com/simpleiot/simpleiot/farmation/islog"
)

func main() {
	flagSim := flag.Bool("sim", false, "Start device simulator")
	flagDiagRun := flag.Bool("diagRun", false, "Run device diagnostics")
	flagDiagList := flag.Bool("diagList", false, "List available diagnostics")
	flagDiagSingle := flag.String("diagSingle", "", "Run single test")
	flagDebugState := flag.Bool("debugState", false, "log state changes")
	flagDebugConfig := flag.Bool("debugConfig", false, "log config changes")
	flagSyslog := flag.Bool("syslog", false, "log to syslog instead of stdout")
	flagDataDir := flag.String("datadir", "", "directory to store data in")
	flagReadPressure := flag.Bool("readPressure", false, "read pressure sensor")
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

	if *flagReadPressure {
		isio.GpioInit()
		ref, sense, err := isio.ReadPressure()
		if err != nil {
			log.Println("Error reading pressure sensor: ", err)
			os.Exit(-1)
		}

		pres := isio.CalcPressure(ref, sense, 250)

		fmt.Printf("Pressure ref: %v, sense: %v, pres: %v\n", ref, sense, pres)
		os.Exit(0)
	}

	if *flagSyslog {
		logwriter, err := islog.Syslog()
		if err == nil {
			log.SetOutput(logwriter)
		} else {
			fmt.Println("Error sending log to syslog: ", err)
		}
	}

	log.Printf("Starting IS app, debug State: %v, debug Config: %v\n", *flagDebugState, *flagDebugConfig)
	if *flagDataDir == "" {
		*flagDataDir = "./"
	}
	app.Run(*flagSim, *flagDebugState, *flagDebugConfig, *flagDataDir)
}
