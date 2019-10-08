package network

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// DebugAtCommands can be set to true to
// debug at commands
var DebugAtCommands = false

// Cmd send a command to modem and read response
// retry 3 times. Port should be a RespReadWriter.
func Cmd(port io.ReadWriter, cmd string) (string, error) {
	var err error

	for try := 0; try < 3; try++ {
		if DebugAtCommands {
			fmt.Println("Modem Tx: ", cmd)
		}

		readString := make([]byte, 100)

		_, err = port.Write([]byte(cmd + "\r"))
		if err != nil {
			continue
		}

		var n int
		n, err = port.Read(readString)

		if err != nil {
			if DebugAtCommands {
				fmt.Println("Modem cmd read error: ", err)
			}
			continue
		}

		readString = readString[:n]

		readStringS := strings.TrimSpace(string(readString))

		if DebugAtCommands {
			fmt.Println("Modem Rx: ", readStringS)
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

// service, rssi, rsrp, sinr, rsrq
// +QCSQ: "CAT-M1",-52,-81,195,-10
var reQcsq = regexp.MustCompile(`\+QCSQ:\s*"(.+)",(-*\d+),(-*\d+),(\d+),(-*\d+)`)

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

		if len(matches) < 6 {
			continue
		}

		found = true

		serviceS := matches[1]
		rssi, _ = strconv.Atoi(matches[2])
		rsrq, _ = strconv.Atoi(matches[3])
		rsrp, _ = strconv.Atoi(matches[5])

		service = serviceS == "CAT-M1"
	}

	if !found {
		err = fmt.Errorf("Error parsing QCSQ response: %v", resp)
	}

	return
}

// service, rssi, rsrp, sinr, rsrq
// +QSPN: "CHN-UNICOM","UNICOM","",0,"46001"
// +QSPN: "Verizon Wireless","VzW","Hologram",0,"311480"
var reQspn = regexp.MustCompile(`\+QSPN:\s*"(.*)","(.*)","(.*)",(\d+),"(.*)"`)

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

		if len(matches) < 6 {
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

// CmdSica is used to send SICA command
func CmdSica(port io.ReadWriter) error {
	return CmdOK(port, "AT^SICA=1,3")
}

// CmdAt just executes a generic at command
func CmdAt(port io.ReadWriter) error {
	return CmdOK(port, "AT")
}
