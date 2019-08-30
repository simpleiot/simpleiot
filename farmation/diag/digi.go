package diag

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Flush any data from port
func Flush(port io.Reader) {

}

// DigiCheckAt puts Digi radio and command mode and executes AT command
func DigiCheckAt(port io.ReadWriter) error {
	commandMode := "+++"

	_, err := port.Write([]byte(commandMode))
	if err != nil {
		return err
	}

	readString := make([]byte, 100)
	n, err := port.Read(readString)
	readString = readString[:n]
	fmt.Print("Digi flush: ", hex.Dump(readString))

	_, err = port.Write([]byte("AT\r"))
	n, err = port.Read(readString)

	readString = readString[:n]
	fmt.Print("Digi read: ", hex.Dump(readString))

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
