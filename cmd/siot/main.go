package main

import (
	"log"
	"os"

	"github.com/simpleiot/simpleiot"
)

func main() {
	if err := simpleiot.StartArgs(os.Args); err != nil {
		log.Println("Error running Simple IoT: ", err)
	}
}
