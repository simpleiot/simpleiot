package main

// eventually, this should become a full fledged client or server test app
// perhaps with an interactive shell.

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/simpleiot/simpleiot/modbus"
	"github.com/simpleiot/simpleiot/respreader"
	"go.bug.st/serial"
)

func usage() {
	fmt.Println("Usage: ")
	flag.PrintDefaults()
	os.Exit(-1)
}

func main() {
	log.Println("modbus client")

	flagPort := flag.String("port", "", "serial port")
	flagBaud := flag.String("baud", "9600", "baud rate")
	flagCount := flag.Int("count", 1, "number of values to read")
	flagReadHoldingRegs := flag.Int("readHoldingRegs", -1, "address to read")
	flagAddress := flag.Int("address", 1, "device address")
	flagFormatInt16 := flag.Bool("int16", false, "Interpret result as 16-bit signed integer")
	flagFormatUInt16 := flag.Bool("uint16", false, "Interpret result as 16-bit unsigned integer")
	flagFormatInt32 := flag.Bool("int32", false, "Interpret result as 32-bit signed integer")
	flagFormatUInt32 := flag.Bool("uint32", false, "Interpret result as 32-bit unsigned integer")
	flagFormatFloat32 := flag.Bool("float32", false, "Interpret result as 32-bit floating point")
	flagScale := flag.Float64("scale", 1, "Scale data by some factor")

	flag.Parse()

	if *flagPort == "" {
		usage()
	}

	baud, err := strconv.Atoi(*flagBaud)

	if err != nil {
		log.Println("Baud rate error: ", err)
		os.Exit(-1)
	}

	mode := &serial.Mode{
		BaudRate: baud,
	}

	port, err := serial.Open(*flagPort, mode)
	if err != nil {
		log.Fatal(err)
	}

	portRR := respreader.NewReadWriteCloser(port, time.Second*1, time.Millisecond*30)
	client := modbus.NewClient(portRR, 1)

	if *flagFormatInt32 || *flagFormatUInt32 || *flagFormatFloat32 {
		*flagCount = *flagCount * 2
	}

	if *flagReadHoldingRegs > 0 {
		log.Printf("Reading holding reg adr: 0x%x, cnt: %v\n",
			*flagReadHoldingRegs, *flagCount)

		regs, err := client.ReadHoldingRegs(byte(*flagAddress),
			uint16(*flagReadHoldingRegs),
			uint16(*flagCount))

		if err != nil {
			log.Println("Error reading holding regs: ", err)
			os.Exit(-1)
		}

		if len(regs) != *flagCount {
			log.Printf("Error, expected %v regs, got %v\n",
				*flagCount, len(regs))
			os.Exit(-1)
		}

		if *flagFormatInt16 {
			values := modbus.RegsToInt16(regs)
			for i, v := range values {
				log.Printf("Value %v: %v\n", i, float64(v)*(*flagScale))
			}
		} else if *flagFormatUInt16 {
			for i, r := range regs {
				log.Printf("Value %v: %v\n", i, float64(r)*(*flagScale))
			}
		} else if *flagFormatInt32 {
			values := modbus.RegsToInt32(regs)
			for i, v := range values {
				log.Printf("Value %v: %v\n", i, float64(v)*(*flagScale))
			}
		} else if *flagFormatUInt32 {
			values := modbus.RegsToUint32(regs)
			for i, v := range values {
				log.Printf("Value %v: %v\n", i, float64(v)*(*flagScale))
			}
		} else if *flagFormatFloat32 {
			values := modbus.RegsToFloat32(regs)
			for i, v := range values {
				log.Printf("Value %v: %v\n", i, float64(v)*(*flagScale))
			}
		} else {
			for i, r := range regs {
				log.Printf("Reg result %v: 0x%x\n", i, r)
			}
		}
	}
}
