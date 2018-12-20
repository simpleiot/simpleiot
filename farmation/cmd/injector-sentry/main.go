package main

import (
	"log"

	"github.com/simpleiot/simpleiot/farmation/isapi"
)

func main() {
	// default action is to start server
	err := isapi.Server()
	if err != nil {
		log.Println("Error starting server: ", err)
	}
}
