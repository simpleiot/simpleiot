package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/particle"
)

func main() {
	flagEvent := flag.String("event", "", "Event to retrieve")
	flag.Parse()

	particleAPIKey := os.Getenv("PARTICLE_API_KEY")
	if particleAPIKey == "" {
		fmt.Println("PARTICLE_API_KEY env var must be set")
		os.Exit(-1)
	}

	err := particle.SampleReader(*flagEvent, particleAPIKey,
		func(id string, samples []data.Sample) {
			fmt.Printf("ID: %v, data: %+v\n", id, samples)
		})

	if err != nil {
		fmt.Println("Get returned error: ", err)
	}
}
