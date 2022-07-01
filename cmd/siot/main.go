package main

import (
	"log"
	"os"

	"github.com/simpleiot/simpleiot/server"
)

func main() {
	if err := server.StartArgs(os.Args); err != nil {
		log.Println("Error running Simple IoT: ", err)
	}
}
