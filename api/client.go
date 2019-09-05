package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/simpleiot/simpleiot/data"
)

// NewSendSamples returns a function that can be used to send samples
// to a SimpleIoT portal instance
func NewSendSamples(portalURL string, debug bool) func(string, []data.Sample) error {
	return func(id string, samples []data.Sample) error {
		sampleURL := portalURL + "/v1/devices/" + id + "/samples"

		tempJSON, err := json.Marshal(samples)
		if err != nil {
			log.Println("Error encoding temp: ", err)
		}

		if debug {
			log.Println("Sending samples: ", tempJSON)
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
