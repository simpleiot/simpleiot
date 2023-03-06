package main

import (
	"fmt"

	"github.com/hashicorp/mdns"
)

func main() {
	// Make a channel for results and start listening
	entriesCh := make(chan *mdns.ServiceEntry, 4)
	go func() {
		for entry := range entriesCh {
			fmt.Printf("Got new entry: %v\n", entry)
		}
	}()

	// Start the lookup
	mdns.Lookup("_http._tcp", entriesCh)
	close(entriesCh)
}
