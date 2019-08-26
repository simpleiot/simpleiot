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

// DeviceSim simulates a simple device
func DeviceSim(portal, deviceID string) {
	log.Printf("starting simulator: ID: %v, portal: %v\n", deviceID, portal)

	sendSample := newSendSample(portal)
	tempSim := NewSim(72, 0.2, 70, 75)
	voltSim := NewSim(2, 0.1, 1, 5)

	for {
		tempSample := data.Sample{
			ID:    "T0",
			Type:  "temp",
			Value: tempSim.Sim(),
		}

		err := sendSample(deviceID, tempSample)
		if err != nil {
			log.Println("Error sending sample: ", err)
		}
		voltSample := data.Sample{
			ID:    "V0",
			Type:  "volt",
			Value: voltSim.Sim(),
		}

		err = sendSample(deviceID, voltSample)
		if err != nil {
			log.Println("Error sending sample: ", err)
		}
		packetDelay()
	}
}
