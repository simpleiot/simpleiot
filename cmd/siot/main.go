package main

import (
	"log"
	"os"

	"github.com/simpleiot/simpleiot/server"
)

func main() {

	if err := server.StartArgs(os.Args); err != nil {
		log.Println("Simple IoT stopped, reason: ", err)
	}
}
