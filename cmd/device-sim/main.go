package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/sim"
)

func main() {
	flagPortal := flag.String("portal", "http://localhost:8080", "Portal URL")
	flagDeviceID := flag.String("deviceId", "1234", "Device ID")
	flagIoID := flag.String("ioId", "A0", "IO ID")
	flag.Parse()

	if *flagPortal == "" {
		fmt.Println("Error: portal url must be set")
		flag.PrintDefaults()
		os.Exit(-1)
	}

	log.Printf("ID: %v, portal: %v\n", *flagDeviceID, *flagPortal)

	tempSim := sim.NewSim(72, 0.2, 70, 75)

	sampleURL := *flagPortal + "/v1/devices/" + *flagDeviceID + "/sample"

	for {
		temp := data.NewSample(*flagIoID, tempSim.Sim())
		tempJSON, err := json.Marshal(temp)
		if err != nil {
			log.Println("Error encoding temp: ", err)
		}

		resp, err := http.Post(sampleURL, "application/json", bytes.NewBuffer(tempJSON))

		if err != nil {
			log.Println("Error posting sample: ", err)
		}

		if resp.StatusCode != http.StatusOK {
			log.Println("Server error: ", resp.Status, sampleURL)
			body, _ := ioutil.ReadAll(resp.Body)
			log.Println("response Body:", string(body))
		}

		defer resp.Body.Close()

		time.Sleep(20 * time.Second)
	}
}
