package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"

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
	flagDebugPortal := flag.Bool("debugPortal", false, "debug portal communication")
	flagDebugModem := flag.Bool("debugModem", false, "enable modem debugging")
	flagSyslog := flag.Bool("syslog", false, "log to syslog instead of stdout")
	flagDataDir := flag.String("datadir", "", "directory to store data in")
	flagReadPressure := flag.Bool("readPressure", false, "read pressure sensor")
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

		log.Printf("Pressure ref: %v, sense: %v, pres: %v\n", ref, sense, pres)
		os.Exit(0)
	}

	/*
			newModem := func() *network.Modem {
				isio.GpioInit()
				p, err := isio.OpenSerialModem()
				if err != nil {
					log.Println("Error opening modem port: ", err)
					os.Exit(-1)
				}

				return network.NewModem(p, "hologram", *flagDebugModem)

			}

		if *flagModemState {
			m := newModem()
			s, err := m.GetState()

			if err != nil {
				log.Println("Error getting modem state: ", err)
				os.Exit(1)
			}

			log.Printf("modem state:\n%v\n", s)
			os.Exit(0)
		}

		if *flagModemSettings {
			m := newModem()
			s, err := m.GetSettings()

			if err != nil {
				log.Println("Error getting modem settings: ", err)
				os.Exit(1)
			}

			log.Printf("modem settings:\n%v\n", s)
			os.Exit(0)
		}

		if *flagModemInfo {
			m := newModem()
			i, err := m.GetInfo()

			if err != nil {
				log.Println("Error getting modem info: ", err)
				os.Exit(1)
			}

			log.Printf("modem info:\n%v\n", i)
			os.Exit(0)
		}

		if *flagModemConfigure {
			m := newModem()
			err := m.Configure()
			if err != nil {
				log.Println("Error configuring modem: ", err)
			}
			log.Println("modem configured")
			os.Exit(0)
		}

		if *flagModemGet {
			m := newModem()
			r, err := m.HTTPGet("http://portal.farmation.us/v1/devices")
			if err != nil {
				log.Println("Error executing GET: ", err)
			}

			log.Println("GET returned: ", string(r))
			os.Exit(0)

		}
	*/

	if *flagModemReset {
		isio.GpioInit()
		isio.ResetModem()
		log.Println("modem reset")
		os.Exit(0)
	}

	if *flagModemStatus {
		isio.GpioInit()
		modem := network.NewModem("bg96", "/dev/ttyUSB2", nil, true)
		status, err := modem.GetStatus()
		if err != nil {
			log.Println("Error getting modem status: ", err)
			os.Exit(-1)
		}

		fmt.Printf("Modem status: %+v\n", status)
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

	if *flagSetIsSN != "" {
		if runtime.GOARCH != "arm" {
			log.Println("Error: Can only set SN on IS")
			os.Exit(-1)
		}

		err := ioutil.WriteFile("/boot/serial-number", []byte(*flagSetIsSN), 600)
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

	params := app.Params{
		Sim:          *flagSim,
		DataDir:      *flagDataDir,
		DebugState:   *flagDebugState,
		DebugConfig:  *flagDebugConfig,
		DebugModem:   *flagDebugModem,
		DebugPortal:  *flagDebugPortal,
		PortalURL:    *flagPortal,
		SerialNumber: *flagSerialNumber,
	}

	app.Run(params)
}
