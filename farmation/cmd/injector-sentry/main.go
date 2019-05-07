package main

import (
	"flag"
	"log"
	"log/syslog"

	"github.com/simpleiot/simpleiot/farmation/app"
	"github.com/simpleiot/simpleiot/farmation/diag"
	"github.com/simpleiot/simpleiot/farmation/isio"
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

	if *flagSyslog {
		logwriter, e := syslog.New(syslog.LOG_NOTICE, "IS")
		if e == nil {
			log.SetOutput(logwriter)
		}
	}

	log.Printf("Starting IS app, debug State: %v, debug Config: %v\n", *flagDebugState, *flagDebugConfig)
	if *flagDataDir == "" {
		*flagDataDir = "./"
	}
	app.Run(*flagSim, *flagDebugState, *flagDebugConfig, *flagDataDir)
}
