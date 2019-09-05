package sim

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/simpleiot/simpleiot/data"
)

func packetDelay() {
	time.Sleep(5 * time.Second)
}

func newSendSamples(portalURL string) func(string, []data.Sample) error {
	return func(id string, samples []data.Sample) error {
		sampleURL := portalURL + "/v1/devices/" + id + "/samples"

		tempJSON, err := json.Marshal(samples)
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

// DeviceSim simulates a simple device
func DeviceSim(portal, deviceID string) {
	log.Printf("starting simulator: ID: %v, portal: %v\n", deviceID, portal)

	sendSamples := newSendSamples(portal)
	tempSim := NewSim(72, 0.2, 70, 75)
	voltSim := NewSim(2, 0.1, 1, 5)

	for {
		samples := make([]data.Sample, 2)
		samples[0] = data.Sample{
			ID:    "T0",
			Type:  "temp",
			Value: tempSim.Sim(),
		}

		samples[1] = data.Sample{
			ID:    "V0",
			Type:  "volt",
			Value: voltSim.Sim(),
		}

		err := sendSamples(deviceID, samples)
		if err != nil {
			log.Println("Error sending samples: ", err)
		}
		packetDelay()
	}
}
