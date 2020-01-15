package isio

import (
	"errors"
	"log"
	"runtime"
	"time"

	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/conn/gpio/gpioreg"
	"periph.io/x/periph/host"
)

// define gpios
const (
	// inputs
	GpioDigitalInjector  string = "GpioDigitalInjector"
	GpioDigitalIrrigator        = "GpioDigitalIrrigator"
	GpioDigitalWaterOn          = "GpioDigitalWaterOn"
	GpioDigitalIn               = "GpioDigitalIn"

	// relay outputs
	GpioRelayInjectorEn = "GpioRelayInjectorEn"
	GpioRelayShutdownEn = "GpioRelayShutdownEn"
	GpioRelayAuxEn      = "GpioRelayAuxEn"

	GpioRelayInjectorFault = "GpioRelayInjectorFault"
	GpioRelayShutdownFault = "GpioRelayShutdownFault"
	GpioRelayAuxFault      = "GpioRelayAuxFault"

	// select between voltage or current loop input
	// high = current loop
	GpioAnalogInSel = "GpioAnalogInSel"
	GpioAuxInSel    = "GpioAuxInSel"

	// status LEDs
	GpioStatusGreen = "GpioStatusGreen"
	GpioStatusRed   = "GpioStatusRed"

	GpioArm = "GpioArm"

	GpioGpsReset = "GpioGpsReset"

	// RS232/RS485 port
	GpioSerialShutdown      = "GpioSerialShutdown"
	GpioSerialLoopback      = "GpioSerialLoopback"
	GpioSerialRsSelectRs485 = "GpioSerialRsSelectRs485"
	GpioSerialRS485RxEn     = "GpioSerialRS485RxEn"
	GpioSerialRs485TxEn     = "GpioSerialRs485TxEn"

	GpioRadioReset = "GpioRadioReset"
	GpioRadioSleep = "GpioRadioSleep"
	//Gpio reset for NL modem is OC, so don't use don't export
	//symbol as we don't want this to be generally used
	//same with modem power on
	gpioModemReset   = "GpioModemReset"
	gpioModemPowerOn = "GpioModemPowerOn"
	GpioModemSleep   = "GpioModemSleep"

	GpioPulseOutput = "GpioPulseOutput"
	GpioFlow1Pulse  = "GpioFlow1Pulse"
	GpioFlow2Pulse  = "GpioFlow2Pulse"

	GpioMainAuxPwr = "GpioMainAuxPwr"
	GpioBackupPwr  = "GpioBackupPwr"

	// LCD
	GpioLcdPinSel = "GpioLcdPinSel"
	GpioLcdReset  = "GpioLcdReset"
	GpioLcdPwm    = "GpioLcdPwm"

	// HWID
	GpioHwID0 = "GpioHwID0"
	GpioHwID1 = "GpioHwID1"
	GpioHwID2 = "GpioHwID2"

	// Regulator valve
	GpioRegValve1 = "GpioRegValve1"
	GpioRegValve2 = "GpioRegValve2"
)

type pin struct {
	Port      string
	ActiveLow bool
	Output    bool
	Pin       gpio.PinIO
}

var pins = map[string]*pin{
	GpioDigitalInjector:  &pin{"PC24", true, false, nil},
	GpioDigitalIrrigator: &pin{"PD8", true, false, nil},
	GpioDigitalWaterOn:   &pin{"PD9", true, false, nil},
	GpioDigitalIn:        &pin{"PD24", true, false, nil},

	GpioRelayInjectorEn: &pin{"PC13", true, true, nil},
	GpioRelayShutdownEn: &pin{"PC9", true, true, nil},
	GpioRelayAuxEn:      &pin{"PB0", true, true, nil},

	GpioRelayInjectorFault: &pin{"PC17", true, false, nil},
	GpioRelayShutdownFault: &pin{"PC12", true, false, nil},
	GpioRelayAuxFault:      &pin{"PC14", true, false, nil},

	GpioAnalogInSel: &pin{"PC15", false, false, nil},
	GpioAuxInSel:    &pin{"PD25", false, false, nil},

	GpioStatusGreen: &pin{"PB6", false, true, nil},
	GpioStatusRed:   &pin{"PB10", false, true, nil},

	GpioArm: &pin{"PC25", true, false, nil},

	GpioGpsReset: &pin{"PA30", true, true, nil},

	GpioSerialShutdown:      &pin{"PB13", true, true, nil},
	GpioSerialLoopback:      &pin{"PB2", false, true, nil},
	GpioSerialRsSelectRs485: &pin{"PB12", false, true, nil},

	GpioRadioReset: &pin{"PB24", true, true, nil},
	GpioRadioSleep: &pin{"PB30", false, true, nil},

	gpioModemReset:   &pin{"PA22", false, false, nil},
	GpioModemSleep:   &pin{"PA27", false, true, nil},
	gpioModemPowerOn: &pin{"PA31", false, false, nil},

	// we are controlling pulse output in kernel now
	//GpioPulseOutput: &pin{"PB7", true, true, nil},
	//GpioFlow1Pulse: &pin{"PB8", true, false, nil},
	//GpioFlow2Pulse: &pin{"PD13", true, false, nil},

	GpioMainAuxPwr: &pin{"PC23", true, false, nil},
	GpioBackupPwr:  &pin{"PD1", true, false, nil},

	// LCD
	GpioLcdPinSel: &pin{"PC5", false, true, nil},
	GpioLcdReset:  &pin{"PC8", true, true, nil},
	GpioLcdPwm:    &pin{"PC3", false, true, nil},

	// HWID
	GpioHwID0: &pin{"PB11", false, false, nil},
	GpioHwID1: &pin{"PC18", false, false, nil},
	GpioHwID2: &pin{"PC16", false, false, nil},
}

// GpioInit is used to initialize gpios
func GpioInit() error {
	// Load periph.io drivers:
	if _, err := host.Init(); err != nil {
		log.Println("Error initializing periph.io host", err)
		return err
	}

	if runtime.GOARCH == "arm" {
		for k, v := range pins {
			pins[k].Pin = gpioreg.ByName(v.Port)
			if pins[k].Pin == nil {
				log.Println("Warning, could init Gpio: ", k, v.Port)
			} else if !v.Output {
				err := pins[k].Pin.In(gpio.PullNoChange, gpio.NoEdge)
				if err != nil {
					log.Println("Error setting pin mode: ", err)
				}
			}
		}

		err := pins[gpioModemReset].Pin.In(gpio.Float, gpio.NoEdge)
		if err != nil {
			log.Println("Error setting pin mode for modem reset: ", err)
		}

		// set modem on signal low to always power on the modem
		err = pins[gpioModemPowerOn].Pin.Out(gpio.Low)
		if err != nil {
			log.Println("Error setting pin mode for modem power on: ", err)
		}

		err = pins[GpioHwID0].Pin.In(gpio.Float, gpio.NoEdge)
		if err != nil {
			log.Println("Error setting pin mode for HwID0: ", err)
		}

		err = pins[GpioHwID1].Pin.In(gpio.Float, gpio.NoEdge)
		if err != nil {
			log.Println("Error setting pin mode for HwID1: ", err)
		}

		err = pins[GpioHwID2].Pin.In(gpio.Float, gpio.NoEdge)
		if err != nil {
			log.Println("Error setting pin mode for HwID2: ", err)
		}
	}

	return nil
}

// GpioOut sets a Gpio value
func GpioOut(name string, value bool) {
	p, ok := pins[name]
	if !ok || p.Pin == nil {
		if runtime.GOARCH == "arm" {
			log.Println("Error setting gpio: ", name)
		}
		return
	}

	p.Pin.Out(gpio.Level(value != p.ActiveLow))
}

// GpioRead reads the current Gpio value
func GpioRead(name string) bool {
	p, ok := pins[name]
	if !ok || p.Pin == nil {
		if runtime.GOARCH == "arm" {
			log.Println("Error reading gpio: ", name)
		}
		return false
	}

	return bool(p.Pin.Read()) != p.ActiveLow
}

// ResetModem resets the modem
// For NL modem, must drive as open collector
func ResetModem() error {
	p, ok := pins[gpioModemReset]
	if !ok || p.Pin == nil {
		if runtime.GOARCH == "arm" {
			return errors.New("Error getting pin")
		}
		return nil
	}

	// drive pin low
	p.Pin.Out(gpio.Low)

	time.Sleep(500 * time.Millisecond)

	// let pin float again
	p.Pin.In(gpio.Float, gpio.NoEdge)

	return nil
}

// GetHwID returns the hardware ID/Version
func GetHwID() int {
	if runtime.GOARCH != "arm" {
		return 99
	}

	ver := 0

	if GpioRead(GpioHwID0) {
		ver += 1 << 0
	}

	if GpioRead(GpioHwID1) {
		ver += 1 << 1
	}

	if GpioRead(GpioHwID2) {
		ver += 1 << 2
	}

	return ver
}
