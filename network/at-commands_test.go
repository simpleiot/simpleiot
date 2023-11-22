package network

import (
	"bytes"
	"testing"
)

// looking for: +CSQ: 9,99
func TestGetSignal(t *testing.T) {
	buf := bytes.NewBufferString("+CSQ: 9,99")

	sig, biterror, err := CmdGetSignal(buf)
	if err != nil {
		t.Fatal("Error: ", err)
	}

	if sig != 29 {
		t.Fatal("Error, signal is: ", sig)
	}

	if biterror != -1 {
		t.Fatal("Error biterror is: ", biterror)
	}
}
