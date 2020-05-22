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

// SampleFilter is used to send samples upstream. It only sends
// the data has changed, and at a max frequency
type SampleFilter struct {
	minSend          time.Duration
	periodicSend     time.Duration
	samples          []data.Sample
	lastSent         time.Time
	lastPeriodicSend time.Time
}

// NewSampleFilter is used to creat a new sample filter
// If samples have changed that get sent out at a minSend interval
// frequency of minSend.
// All samples are periodically sent at lastPeriodicSend interval.
// Set minSend to 0 for things like config settings where you want them
// to be sent whenever anything changes.
func NewSampleFilter(minSend, periodicSend time.Duration) *SampleFilter {
	return &SampleFilter{
		minSend:      minSend,
		periodicSend: periodicSend,
	}
}

// returns true if sample has changed, and merges sample with saved samples
func (sf *SampleFilter) add(sample data.Sample) bool {
	for i, s := range sf.samples {
		if sample.ID == s.ID && sample.Type == s.Type {
			if sample.Value == s.Value {
				return false
			}

			sf.samples[i].Value = sample.Value
			return true
		}
	}

	// sample not found, add to array
	sf.samples = append(sf.samples, sample)
	return true
}

// Add adds samples and returns samples that meet the filter criteria
func (sf *SampleFilter) Add(samples []data.Sample) []data.Sample {
	if time.Since(sf.lastPeriodicSend) > sf.periodicSend {
		// send all samples
		for _, s := range samples {
			sf.add(s)
		}

		sf.lastPeriodicSend = time.Now()
		sf.lastSent = sf.lastPeriodicSend
		return sf.samples
	}

	if sf.minSend != 0 && time.Since(sf.lastSent) < sf.minSend {
		// don't return anything as
		return []data.Sample{}
	}

	// now check if anything has changed and just send what has changed
	// only
	var ret []data.Sample

	for _, s := range samples {
		if sf.add(s) {
			ret = append(ret, s)
		}
	}

	if len(ret) > 0 {
		sf.lastSent = time.Now()
	}

	return ret
}
