package main

import (
	"flag"
	"fmt"

	"github.com/simpleiot/simpleiot/particle"
)

func main() {
	flagAPIKey := flag.String("apikey", "", "Particle API key")
	flagEvent := flag.String("event", "", "Event to retrieve")
	flag.Parse()

	err := particle.SampleReader(*flagEvent, *flagAPIKey, func(data []byte) {
		fmt.Println("data: ", string(data))
	})

	if err != nil {
		fmt.Println("Get returned error: ", err)
	}
}
