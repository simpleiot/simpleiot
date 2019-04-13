package diag

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	nbreader "github.com/svent/go-nbreader"
)

// Flush any data from port
func Flush(port io.Reader) {

}

// DigiCheckAt puts Digi radio and command mode and executes AT command
func DigiCheckAt(port io.ReadWriter) error {
	fmt.Println("CLIFF: DigiCheckAt")
	portTo := nbreader.NewNBReader(port, 100, nbreader.Timeout(500*time.Millisecond))
	commandMode := "+++"

	_, err := port.Write([]byte(commandMode))
	if err != nil {
		return err
	}

	time.Sleep(1100 * time.Millisecond)
	// flush any data
	readString := make([]byte, 100)
	n, err := portTo.Read(readString)
	readString = readString[:n]
	fmt.Println("cliff flush: ", hex.Dump(readString))

	_, err = port.Write([]byte("AT\n"))
	time.Sleep(200 * time.Millisecond)
	n, err = portTo.Read(readString)

	readString = readString[:n]
	fmt.Println("cliff readString: ", hex.Dump(readString))

	if err != nil {
		return err
	}

	readStringS := strings.TrimSpace(string(readString))
	if string(readStringS) != "OK" {
		fmt.Println("readString: ", readStringS)
		return errors.New("Expected OK, got: " + readStringS)
	}

	return nil
}
