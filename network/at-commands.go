package network

import (
	"errors"
	"fmt"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"
)

// DebugAtCommands can be set to true to
// debug at commands
var DebugAtCommands = false

// RsrqValueToDb converts AT value to dB
func RsrqValueToDb(value int) float64 {
	if value < 0 || value > 34 {
		// unknown value
		return 0
	}

	return -20 + float64(value)*0.5
}

// RsrpValueToDb converts AT value to dB
func RsrpValueToDb(value int) float64 {
	if value < 0 || value > 97 {
		// unknown value
		return 0
	}

	return -141 + float64(value)
}

// Cmd send a command to modem and read response
// retry 3 times. Port should be a RespReadWriter.
func Cmd(port io.ReadWriter, cmd string) (string, error) {
	var err error

	for try := 0; try < 3; try++ {
		if DebugAtCommands {
			log.Println("Modem Tx:", cmd)
		}

		readString := make([]byte, 512)

		_, err = port.Write([]byte(cmd + "\r"))
		if err != nil {
			if DebugAtCommands {
				log.Println("Modem cmd write error:", err)
			}
			continue
		}

		var n int
		n, err = port.Read(readString)

		if err != nil {
			if DebugAtCommands {
				log.Println("Modem cmd read error:", err)
			}
			continue
		}

		readString = readString[:n]

		readStringS := strings.TrimSpace(string(readString))

		if DebugAtCommands {
			log.Println("Modem Rx:", readStringS)
		}

		return readStringS, nil
	}

	return "", err
}

// CmdOK runs the command and checks for OK response
func CmdOK(port io.ReadWriter, cmd string) error {
	resp, err := Cmd(port, cmd)
	if err != nil {
		return err
	}

	return checkRespOK(resp)
}

var errorNoOK = errors.New("command did not return OK")

func checkRespOK(resp string) error {
	for _, line := range strings.Split(string(resp), "\n") {
		if strings.Contains(line, "OK") {
			return nil
		}
	}

	return errorNoOK
}

// +CESQ: 99,99,255,255,13,34
// +CESQ: 99,99,255,255,17,42
// +CESQ: rxlen,ber,rscp,ecno,rsrq,rsrp
var reCesq = regexp.MustCompile(`\+CESQ:\s*(\d+),(\d+),(\d+),(\d+),(\d+),(\d+)`)

// CmdCesq is used to send the AT+CESQ command
func CmdCesq(port io.ReadWriter) (rsrq, rsrp int, err error) {
	var resp string
	resp, err = Cmd(port, "AT+CESQ")
	if err != nil {
		return
	}

	found := false

	for _, line := range strings.Split(string(resp), "\n") {
		matches := reCesq.FindStringSubmatch(line)

		if len(matches) >= 6 {
			rsrqI, _ := strconv.Atoi(matches[5])
			rsrpI, _ := strconv.Atoi(matches[6])

			rsrq = int(RsrqValueToDb(rsrqI))
			rsrp = int(RsrpValueToDb(rsrpI))

			return
		}
	}

	if !found {
		err = fmt.Errorf("Error parsing CESQ response: %v", resp)
	}

	return
}

// service, rssi, rsrp, sinr, rsrq
// +QCSQ: "NOSERVICE"
// +QCSQ: "GSM",-69
// +QCSQ: "CAT-M1",-52,-81,195,-10
var reQcsq = regexp.MustCompile(`\+QCSQ:\s*"(.+)"`)
var reQcsqM1 = regexp.MustCompile(`\+QCSQ:\s*"(.+)",(-*\d+),(-*\d+),(\d+),(-*\d+)`)

// CmdQcsq is used to send the AT+QCSQ command
func CmdQcsq(port io.ReadWriter) (service bool, rssi, rsrp, rsrq int, err error) {
	var resp string
	resp, err = Cmd(port, "AT+QCSQ")
	if err != nil {
		return
	}

	found := false

	for _, line := range strings.Split(string(resp), "\n") {
		matches := reQcsq.FindStringSubmatch(line)

		if len(matches) < 2 {
			continue
		}

		found = true

		serviceS := matches[1]

		matches = reQcsqM1.FindStringSubmatch(line)

		if len(matches) >= 6 {
			rssi, _ = strconv.Atoi(matches[2])
			rsrp, _ = strconv.Atoi(matches[3])
			rsrq, _ = strconv.Atoi(matches[5])
		}

		service = serviceS == "CAT-M1"
	}

	if !found {
		err = fmt.Errorf("Error parsing QCSQ response: %v", resp)
	}

	return
}

// possible return values
// service, rssi, rsrp, sinr, rsrq
// ERROR (if no connection)
// +QSPN: "CHN-UNICOM","UNICOM","",0,"46001"
// +QSPN: "Verizon Wireless","VzW","Hologram",0,"311480"
var reQspn = regexp.MustCompile(`\+QSPN:\s*"(.*)"`)

// CmdQspn is used to send the AT+QSPN command
func CmdQspn(port io.ReadWriter) (network string, err error) {
	var resp string
	resp, err = Cmd(port, "AT+QSPN")
	if err != nil {
		return
	}

	found := false

	for _, line := range strings.Split(string(resp), "\n") {
		matches := reQspn.FindStringSubmatch(line)

		if len(matches) < 2 {
			continue
		}

		found = true

		network = matches[1]
	}

	if !found {
		err = fmt.Errorf("Error parsing QSPN response: %v", resp)
	}

	return
}

// CmdSetApn is used to set the APN using the GCDCONT command
func CmdSetApn(port io.ReadWriter, apn string) error {
	return CmdOK(port, "AT+CGDCONT=3,\"IPV4V6\",\""+apn+"\"")
}

// CmdFunMin sets the modem functionality to min
func CmdFunMin(port io.ReadWriter) error {
	return CmdOK(port, "AT+CFUN=0")
}

// CmdFunFull sets the modem functionality to full
func CmdFunFull(port io.ReadWriter) error {
	return CmdOK(port, "AT+CFUN=1")
}

// CmdAttach attaches modem to network
func CmdAttach(port io.ReadWriter) error {
	return CmdOK(port, "AT+CGATT=1")
}

// CmdSica is used to send SICA command
func CmdSica(port io.ReadWriter) error {
	return CmdOK(port, "AT^SICA=1,3")
}

// CmdAt just executes a generic at command
func CmdAt(port io.ReadWriter) error {
	return CmdOK(port, "AT")
}

// BG96MAR02A07M1G_01.007.01.007
var reCmdVersionBg96 = regexp.MustCompile(`BG96.*`)

// CmdGetFwVersionBG96 gets FW version from BG96 modem
func CmdGetFwVersionBG96(port io.ReadWriter) (string, error) {
	resp, err := Cmd(port, "AT+CGMR")
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(resp, "\n") {
		match := reCmdVersionBg96.FindString(line)
		if match != "" {
			return match, nil
		}
	}

	return "", fmt.Errorf("Error parsing AT+CGMR response: %v", resp)
}

// REVISION 4.3.1.0c
var reATI = regexp.MustCompile(`REVISION (\S+)`)

// CmdATI gets version # from modem
func CmdATI(port io.ReadWriter) (string, error) {
	resp, err := Cmd(port, "ATI")
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(resp, "\n") {
		matches := reATI.FindStringSubmatch(line)
		if len(matches) >= 2 {
			return matches[1], nil
		}
	}

	return "", fmt.Errorf("Error parsing ATI response: %v", resp)
}

var reUsbCfg = regexp.MustCompile(`USBCFG:\s*(\d+)`)

// CmdGetUsbCfg gets the USB config. For Telit modems, 0 is ppp, 3 is USB network
func CmdGetUsbCfg(port io.ReadWriter) (int, error) {
	resp, err := Cmd(port, "AT#USBCFG?")
	if err != nil {
		return -1, err
	}

	for _, line := range strings.Split(string(resp), "\n") {
		matches := reUsbCfg.FindStringSubmatch(line)
		if len(matches) >= 2 {
			cfg, _ := strconv.Atoi(matches[1])

			return cfg, nil
		}
	}

	return -1, fmt.Errorf("Error parsing response of USBCFG")
}

// CmdSetUsbConfig is configures the USB mode of Telit modems
// 0 = ppp
// 3 = usb network
func CmdSetUsbConfig(port io.ReadWriter, cfg int) error {
	return CmdOK(port, fmt.Sprintf("AT#USBCFG=%v", cfg))
}

var reApn = regexp.MustCompile(`CGDCONT: 3,"IPV4V6","(.*?)"`)

// CmdGetApn gets the APN
func CmdGetApn(port io.ReadWriter) (string, error) {
	resp, err := Cmd(port, "AT+CGDCONT?")
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(resp, "\n") {
		matches := reApn.FindStringSubmatch(line)
		if len(matches) >= 2 {
			apn := matches[1]
			return apn, nil
		}
	}

	return "", fmt.Errorf("Error parsing AT+CGDCONT?: %v", resp)
}

var reFwSwitch = regexp.MustCompile(`FWSWITCH:\s*(\d)`)

// CmdGetFwSwitch returns the firmware used in Telit modems
// 0 - AT&T
// 1 - Verizon
func CmdGetFwSwitch(port io.ReadWriter) (int, error) {
	resp, err := Cmd(port, "AT#FWSWITCH?")
	if err != nil {
		return -1, err
	}

	for _, line := range strings.Split(resp, "\n") {
		matches := reFwSwitch.FindStringSubmatch(line)
		if len(matches) >= 2 {
			fw, _ := strconv.Atoi(matches[1])
			return fw, nil
		}
	}

	return -1, fmt.Errorf("Error parsing AT#FWSWITCH?: %v", resp)
}

// GpioDir specifies Gpio Direction
type GpioDir int

// Gpio dir values
const (
	GpioDirUnknown GpioDir = -1
	GpioIn         GpioDir = 0
	GpioOut        GpioDir = 1
)

// GpioLevel describes GPIO level
type GpioLevel int

// Gpio Level values
const (
	GpioLevelUnknown GpioLevel = -1
	GpioLow          GpioLevel = 0
	GpioHigh         GpioLevel = 1
)

// #GPIO: 0,0,4
var reGpio = regexp.MustCompile(`GPIO:\s*(\d+),(\d+)`)

// CmdGetGpio is used to get GPIO state on Telit modems
func CmdGetGpio(port io.ReadWriter, gpio int) (GpioDir, GpioLevel, error) {
	cmd := fmt.Sprintf("AT#GPIO=%v,2", gpio)
	resp, err := Cmd(port, cmd)
	if err != nil {
		return GpioDirUnknown, GpioLevelUnknown, err
	}

	for _, line := range strings.Split(resp, "\n") {
		matches := reGpio.FindStringSubmatch(line)
		if len(matches) >= 3 {
			dir, _ := strconv.Atoi(matches[1])
			level, _ := strconv.Atoi(matches[2])
			return GpioDir(dir), GpioLevel(level), nil
		}
	}

	return GpioDirUnknown, GpioLevelUnknown, fmt.Errorf("Error parsing AT#GPIO: %v", resp)
}

// CmdSetGpio is used to set GPIO state in Telit modems
func CmdSetGpio(port io.ReadWriter, gpio int, level GpioLevel) error {
	err := CmdOK(port, "AT+CFUN=4")
	if err != nil {
		return err
	}
	cmd := fmt.Sprintf("AT#GPIO=%v,%v,1,1", gpio, level)
	err = CmdOK(port, cmd)
	if err != nil {
		return err
	}
	return CmdOK(port, "AT+CFUN=1")
}

// looking for: +CSQ: 9,99
var reSig = regexp.MustCompile(`\+CSQ:\s*(\d+),(\d+)`)

// CmdGetSignal gets signal strength
func CmdGetSignal(port io.ReadWriter) (int, int, error) {
	resp, err := Cmd(port, "AT+CSQ")
	if err != nil {
		return 0, 0, err
	}

	for _, line := range strings.Split(resp, "\n") {
		matches := reSig.FindStringSubmatch(line)
		if len(matches) >= 3 {
			signalStrengthF, _ := strconv.ParseFloat(matches[1], 32)
			bitErrorRateF, _ := strconv.ParseFloat(matches[2], 32)

			var signalStrength, bitErrorRate int

			// normalize numbers and return -1 if not known
			if signalStrengthF == 99 {
				signalStrength = -1
			} else {
				signalStrength = int(signalStrengthF * 100 / 31)
			}

			if bitErrorRateF == 99 {
				bitErrorRate = -1
			} else {
				bitErrorRate = int(bitErrorRateF * 100 / 7)
			}

			return signalStrength, bitErrorRate, nil
		}
	}

	return 0, 0, fmt.Errorf("Error parsing AT+CSQ response: %v", resp)
}

// +CNUM: "Line 1","+15717759540",145
// +CNUM: "","18167882915",129
var reCmdPhoneNum = regexp.MustCompile(`(\d{11,})`)

// CmdGetPhoneNum gets phone number from modem
func CmdGetPhoneNum(port io.ReadWriter) (string, error) {
	resp, err := Cmd(port, "AT+CNUM")
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(resp, "\n") {
		matches := reCmdPhoneNum.FindStringSubmatch(line)
		if len(matches) >= 2 {
			return matches[1], nil
		}
	}

	return "", fmt.Errorf("Error parsing AT+CNUM response: %v", resp)
}

// +CCID: "89148000000637720260",""
// +ICCID: 8901260881206806423
var reCmdSim = regexp.MustCompile(`(\d{19,})`)

// CmdGetSim gets SIM # from modem
func CmdGetSim(port io.ReadWriter) (string, error) {
	resp, err := Cmd(port, "AT+CCID?")
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(resp, "\n") {
		matches := reCmdSim.FindStringSubmatch(line)
		if len(matches) >= 2 {
			return matches[1], nil
		}
	}

	return "", fmt.Errorf("Error parsing AT+CCID? response: %v", resp)
}

// 356278070013083
var reCmdImei = regexp.MustCompile(`(\d{15,})`)

// CmdGetImei gets IMEI # from modem
func CmdGetImei(port io.ReadWriter) (string, error) {
	resp, err := Cmd(port, "AT+CGSN")
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(resp, "\n") {
		matches := reCmdImei.FindStringSubmatch(line)
		if len(matches) >= 2 {
			return matches[1], nil
		}
	}

	return "", fmt.Errorf("Error parsing AT+CGSN response: %v", resp)
}

// CmdGetSimBg96 returns SIM for bg96 modems
func CmdGetSimBg96(port io.ReadWriter) (string, error) {
	resp, err := Cmd(port, "AT+QCCID")
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(resp, "\n") {
		matches := reCmdSim.FindStringSubmatch(line)
		if len(matches) >= 2 {
			return matches[1], nil
		}
	}

	return "", fmt.Errorf("Error parsing AT+QCCID response: %v", resp)
}

// +QGPSGNMEA: $GPGGA,,,,,,0,,,,,,,,*66
var reQGPSNEMA = regexp.MustCompile(`\+QGPSGNMEA:\s*(.*)`)

// CmdGGA gets GPS information from modem
func CmdGGA(port io.ReadWriter) (string, error) {
	resp, err := Cmd(port, "AT+QGPSGNMEA=\"GGA\"")
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(resp, "\n") {
		matches := reQGPSNEMA.FindStringSubmatch(line)
		if len(matches) >= 2 {
			return matches[1], nil
		}
	}

	return "", fmt.Errorf("Error parsing AT+QGPSGNMEA response: %v", resp)
}

// CmdBg96ForceLTE forces BG96 modems to use LTE only, (no 2G)
func CmdBg96ForceLTE(port io.ReadWriter) error {
	return CmdOK(port, "AT+QCFG=\"nwscanmode\",3,1")
}

// BG96ScanMode is a type that defines the varios BG96 scan modes
type BG96ScanMode int

// valid scan modes
const (
	BG96ScanModeUnknown BG96ScanMode = -1
	BG96ScanModeAuto    BG96ScanMode = 0
	BG96ScanModeGSM     BG96ScanMode = 1
	BG96ScanModeLTE     BG96ScanMode = 3
)

// +QCFG: "nwscanmode",3
var reBg96ScanMode = regexp.MustCompile(`\++QCFG: "nwscanmode",(\d)`)

// CmdBg96GetScanMode returns the current modem scan mode
func CmdBg96GetScanMode(port io.ReadWriter) (BG96ScanMode, error) {
	resp, err := Cmd(port, "AT+QCFG=\"nwscanmode\"")
	if err != nil {
		return BG96ScanModeUnknown, err
	}

	for _, line := range strings.Split(resp, "\n") {
		matches := reBg96ScanMode.FindStringSubmatch(line)
		if len(matches) >= 2 {
			mode, err := strconv.Atoi(matches[1])
			if err != nil {
				continue
			}

			return BG96ScanMode(mode), nil
		}
	}

	return BG96ScanModeUnknown,
		fmt.Errorf("Error parsing AT+QGPSGNMEA response: %v", resp)
}

// TODO, add AT+COPS command to get current carrier
// AT+COPS?
// +COPS: 0 (no connection)
// +COPS: 0,0,"AT&T Hologram",8
var reCops = regexp.MustCompile(`\+COPS:`)
var reCopsCon = regexp.MustCompile(`\+COPS:\s*(.*),(.*),"(.+)"`)

// CmdCops is used determine what carrier we are connected to
func CmdCops(port io.ReadWriter) (carrier string, err error) {
	var resp string
	resp, err = Cmd(port, "AT+COPS?")
	if err != nil {
		return
	}

	found := false

	for _, line := range strings.Split(string(resp), "\n") {
		if reCops.FindStringIndex(line) == nil {
			continue
		}

		found = true

		matches := reCopsCon.FindStringSubmatch(line)

		if len(matches) >= 4 {
			carrier = matches[3]
		}
	}

	if !found {
		err = fmt.Errorf("Error parsing COPS? response: %v", resp)
	}

	return
}

// CmdReboot reboots modem
func CmdReboot(port io.ReadWriter) error {
	return CmdOK(port, "AT+CFUN=1,1")
}
