package api

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

// NewGetCmd returns a function that can be used to get device commands from the
// portal.
func NewGetCmd(portalURL, deviceID string, timeout time.Duration, debug bool) func() (data.DeviceCmd, error) {
	var netClient = &http.Client{
		Timeout: timeout,
	}

	return func() (data.DeviceCmd, error) {
		sampleURL := portalURL + "/v1/devices/" + deviceID + "/cmd"

		resp, err := netClient.Get(sampleURL)

		var cmd data.DeviceCmd

		if err != nil {
			return cmd, err
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errstring := "Server error: " + resp.Status + " " + sampleURL
			body, _ := ioutil.ReadAll(resp.Body)
			errstring += " " + string(body)
			return cmd, errors.New(errstring)
		}

		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(&cmd)
		if err != nil {
			return cmd, err
		}

		if debug && cmd.Cmd != "" {
			log.Printf("Got cmd: %+v\n", cmd)
		}

		return cmd, nil
	}
}

// NewSendSamples returns a function that can be used to send samples
// to a SimpleIoT portal instance
func NewSendSamples(portalURL, deviceID string, timeout time.Duration, debug bool) func([]data.Sample) error {
	var netClient = &http.Client{
		Timeout: timeout,
	}

	return func(samples []data.Sample) error {
		sampleURL := portalURL + "/v1/devices/" + deviceID + "/samples"

		tempJSON, err := json.Marshal(samples)
		if err != nil {
			log.Println("Error encoding temp: ", err)
		}

		if debug {
			log.Println("Sending samples: ", string(tempJSON))
		}

		resp, err := netClient.Post(sampleURL, "application/json", bytes.NewBuffer(tempJSON))

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

// NewSetVersion sets the device version in the portal
func NewSetVersion(portalURL, deviceID string, timeout time.Duration, debug bool) func(ver data.DeviceVersion) error {
	var netClient = &http.Client{
		Timeout: timeout,
	}

	return func(ver data.DeviceVersion) error {
		sampleURL := portalURL + "/v1/devices/" + deviceID + "/version"

		tempJSON, err := json.Marshal(ver)
		if err != nil {
			log.Println("Error encoding temp: ", err)
		}

		if debug {
			log.Println("Sending samples: ", string(tempJSON))
		}

		resp, err := netClient.Post(sampleURL, "application/json", bytes.NewBuffer(tempJSON))

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
