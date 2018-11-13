package main

import (
	"bytes"
	"encoding/json"
	"errors"
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

func packetDelay() {
	time.Sleep(5 * time.Second)
}

func newSendSample(portalURL string) func(string, data.Sample) error {
	return func(id string, sample data.Sample) error {
		sampleURL := portalURL + "/v1/devices/" + id + "/sample"

		tempJSON, err := json.Marshal(sample)
		if err != nil {
			log.Println("Error encoding temp: ", err)
		}

		resp, err := http.Post(sampleURL, "application/json", bytes.NewBuffer(tempJSON))

		if err != nil {
			return err
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errstring := "Server error: " + resp.Status + " " + sampleURL
			body, _ := ioutil.ReadAll(resp.Body)
			errstring += " " + string(body)
			return errors.New(errstring)
		}

		return nil
	}
}

func main() {
	flagPortal := flag.String("portal", "http://localhost:8080", "Portal URL")
	flagDeviceID := flag.String("deviceId", "1234", "Device ID")
	flag.Parse()

	if *flagPortal == "" {
		fmt.Println("Error: portal url must be set")
		flag.PrintDefaults()
		os.Exit(-1)
	}

	log.Printf("ID: %v, portal: %v\n", *flagDeviceID, *flagPortal)

	sendSample := newSendSample(*flagPortal)
	tempSim := sim.NewSim(72, 0.2, 70, 75)
	voltSim := sim.NewSim(2, 0.1, 1, 5)

	for {
		tempSample := data.NewSample("T0", tempSim.Sim())
		err := sendSample(*flagDeviceID, tempSample)
		if err != nil {
			log.Println("Error sending sample: ", err)
		}
		voltSample := data.NewSample("V0", voltSim.Sim())
		err = sendSample(*flagDeviceID, voltSample)
		if err != nil {
			log.Println("Error sending sample: ", err)
		}
		packetDelay()
	}
}
