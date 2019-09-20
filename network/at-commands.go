package network

import (
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
		readString := make([]byte, 100)

		_, err = port.Write([]byte(cmd + "\r"))
		if err != nil {
			continue
		}

		var n int
		n, err = port.Read(readString)

		if err != nil {
			continue
		}

		readString = readString[:n]

		readStringS := strings.TrimSpace(string(readString))

		if DebugAtCommands {
			fmt.Printf("Modem: %v -> %v\n", cmd, readStringS)
		}

		return readStringS, nil
	}

	return "", err
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
