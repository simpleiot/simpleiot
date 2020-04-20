package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/tbrandon/mbserver"
	"go.bug.st/serial"
)

func usage() {
	fmt.Println("Usage: ")
	flag.PrintDefaults()
	os.Exit(-1)
}

func main() {
	log.Println("modbus simulator")

	port := flag.String("port", "", "serial port")

	flag.Parse()

	if *port == "" {
		usage()
	}

	log.Println("Starting server on: ", *port)

	serv := mbserver.NewServer(50, 50, 50, 50)
	serv.Debug = true

	// Override ReadDiscreteInputs function.
	serv.RegisterFunctionHandler(2,
		func(s *mbserver.Server, frame mbserver.Framer) ([]byte, *mbserver.Exception) {
			log.Println("register handler")

			frameData := frame.GetData()
			register := int(binary.BigEndian.Uint16(frameData[0:2]))
			numRegs := int(binary.BigEndian.Uint16(frameData[2:4]))
			endRegister := register + numRegs

			log.Printf("%v\n", register)
			log.Printf("%v\n", numRegs)
			log.Printf("%v\n", endRegister)

			// Check the request is within the allocated memory
			if endRegister > 65535 {
				return []byte{}, &mbserver.IllegalDataAddress
			}
			dataSize := numRegs / 8
			if (numRegs % 8) != 0 {
				dataSize++
			}
			data := make([]byte, 1+dataSize)
			data[0] = byte(dataSize)
			for i := range s.DiscreteInputs[register:endRegister] {
				// Return all 1s, regardless of the value in the DiscreteInputs array.
				shift := uint(i) % 8
				data[1+i/8] |= byte(1 << shift)
			}

			return data, &mbserver.Success
		})

	/*
		err := serv.ListenRTU(&serial.Config{
			Address:  *port,
			BaudRate: 115200,
			DataBits: 8,
			StopBits: 1,
			Parity:   "N",
			Timeout:  10 * time.Second})
	*/

	err := serv.ListenRTU(*port, &serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	})

	if err != nil {
		log.Println("Error opening modbus port: ", err)
	}

	defer serv.Close()

	select {}

}
