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
	"github.com/simpleiot/simpleiot/network"
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
	flagModemState := flag.Bool("modemState", false, "read modem state")
	flagModemSettings := flag.Bool("modemSettings", false, "read modem settings")
	flagModemInfo := flag.Bool("modemInfo", false, "read modem info")
	flagModemDebug := flag.Bool("modemDebug", false, "enable modem debugging")
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

	if *flagModemState {
		isio.GpioInit()
		p, err := isio.OpenSerialModem()
		if err != nil {
			log.Println("Error opening modem port: ", err)
			os.Exit(-1)
		}

		m := network.NewModem(p, *flagModemDebug)

		s, err := m.GetState()

		if err != nil {
			log.Println("Error getting modem state: ", err)
			os.Exit(1)
		}

		fmt.Printf("modem state:\n%v\n", s)
		os.Exit(0)
	}

	if *flagModemSettings {
		isio.GpioInit()
		p, err := isio.OpenSerialModem()
		if err != nil {
			log.Println("Error opening modem port: ", err)
			os.Exit(-1)
		}

		m := network.NewModem(p, *flagModemDebug)

		s, err := m.GetSettings()

		if err != nil {
			log.Println("Error getting modem settings: ", err)
			os.Exit(1)
		}

		fmt.Printf("modem settings:\n%v\n", s)
		os.Exit(0)
	}

	if *flagModemInfo {
		isio.GpioInit()
		p, err := isio.OpenSerialModem()
		if err != nil {
			log.Println("Error opening modem port: ", err)
			os.Exit(-1)
		}

		m := network.NewModem(p, *flagModemDebug)

		i, err := m.GetInfo()

		if err != nil {
			log.Println("Error getting modem info: ", err)
			os.Exit(1)
		}

		fmt.Printf("modem info:\n%v\n", i)
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

	params := app.Params{
		Sim:         *flagSim,
		DataDir:     *flagDataDir,
		DebugState:  *flagDebugState,
		DebugConfig: *flagDebugConfig,
		DebugModem:  *flagModemDebug,
	}

	app.Run(params)
}
