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
func NewGetCmd(portalURL, deviceID string, timeout time.Duration, debug bool) func() (data.NodeCmd, error) {
	var netClient = &http.Client{
		Timeout: timeout,
	}

	return func() (data.NodeCmd, error) {
		sampleURL := portalURL + "/v1/devices/" + deviceID + "/cmd"

		resp, err := netClient.Get(sampleURL)

		var cmd data.NodeCmd

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

// Sample is a custom value of data.Sample with Time set to a pointer. This allows
// omitempty to work for zero timestamps to avoid bloating JSON packets.
type Sample struct {
	// Type of sample (voltage, current, key, etc)
	Type string `json:"type,omitempty" influx:"type,tag"`

	// ID of the device that provided the sample
	ID string `json:"id,omitempty" influx:"id,tag"`

	// Average OR
	// Instantaneous analog or digital value of the sample.
	// 0 and 1 are used to represent digital values
	Value float64 `json:"value,omitempty" influx:"value"`

	// statistical values that may be calculated
	Min float64 `json:"min,omitempty" influx:"min"`
	Max float64 `json:"max,omitempty" influx:"max"`

	// Time the sample was taken
	Time *time.Time `json:"time,omitempty" boltholdKey:"Time" gob:"-" influx:"time"`

	// Duration over which the sample was taken
	Duration time.Duration `json:"duration,omitempty" influx:"duration"`

	// Tags are additional attributes used to describe the sample
	// You might add things like friendly name, etc.
	Tags map[string]string `json:"tags,omitempty" influx:"-"`

	// Attributes are additional numerical values
	Attributes map[string]float64 `json:"attributes,omitempty" influx:"-"`
}

// NewSample converts a data.Sample to Sample and rounds floating point
// values to 3 dec places.
func NewSample(s data.Sample) Sample {
	var time *time.Time

	if !s.Time.IsZero() {
		time = &s.Time
	}

	return Sample{
		Type:       s.Type,
		ID:         s.ID,
		Value:      s.Value,
		Min:        s.Min,
		Max:        s.Max,
		Time:       time,
		Tags:       s.Tags,
		Attributes: s.Attributes,
	}
}

// NewSamples converts []data.Sample to []Sample
func NewSamples(samples []data.Sample) []Sample {
	ret := make([]Sample, len(samples))

	for i, s := range samples {
		ret[i] = NewSample(s)
	}

	return ret
}

// NewSendSamples returns a function that can be used to send samples
// to a SimpleIoT portal instance
func NewSendSamples(portalURL, deviceID string, timeout time.Duration, debug bool) func([]data.Sample) error {
	var netClient = &http.Client{
		Timeout: timeout,
	}

	return func(samples []data.Sample) error {
		sampleURL := portalURL + "/v1/devices/" + deviceID + "/samples"

		tempJSON, err := json.Marshal(NewSamples(samples))
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
func NewSetVersion(portalURL, deviceID string, timeout time.Duration, debug bool) func(ver data.NodeVersion) error {
	var netClient = &http.Client{
		Timeout: timeout,
	}

	return func(ver data.NodeVersion) error {
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

// Point is a custom value of data.Point with Time set to a pointer. This allows
// omitempty to work for zero timestamps to avoid bloating JSON packets.
type Point struct {
	// Type of sample (voltage, current, key, etc)
	Type string `json:"type,omitempty" influx:"type,tag"`

	// ID of the device that provided the sample
	ID string `json:"id,omitempty" influx:"id,tag"`

	// Average OR
	// Instantaneous analog or digital value of the sample.
	// 0 and 1 are used to represent digital values
	Value float64 `json:"value,omitempty" influx:"value"`

	// statistical values that may be calculated
	Min float64 `json:"min,omitempty" influx:"min"`
	Max float64 `json:"max,omitempty" influx:"max"`

	// Time the sample was taken
	Time *time.Time `json:"time,omitempty" boltholdKey:"Time" gob:"-" influx:"time"`

	// Duration over which the sample was taken
	Duration time.Duration `json:"duration,omitempty" influx:"duration"`
}

// NewPoint converts a data.Point to Point and rounds floating point
// values to 3 dec places.
func NewPoint(s data.Point) Point {
	var time *time.Time

	if !s.Time.IsZero() {
		time = &s.Time
	}

	return Point{
		Type:  s.Type,
		ID:    s.ID,
		Value: s.Value,
		Min:   s.Min,
		Max:   s.Max,
		Time:  time,
	}
}

// NewPoints converts []data.Sample to []Sample
func NewPoints(points []data.Point) []Point {
	ret := make([]Point, len(points))

	for i, p := range points {
		ret[i] = NewPoint(p)
	}

	return ret
}

// NewSendPoints returns a function that can be used to send points
// to a SimpleIoT portal instance
func NewSendPoints(portalURL, deviceID, authToken string, timeout time.Duration, debug bool) func(data.Points) error {
	var netClient = &http.Client{
		Timeout: timeout,
	}

	return func(points data.Points) error {
		sampleURL := portalURL + "/v1/devices/" + deviceID + "/points"

		tempJSON, err := json.Marshal(NewPoints(points))
		if err != nil {
			return err
		}

		if debug {
			log.Println("Sending samples: ", string(tempJSON))
		}

		req, err := http.NewRequest("POST", sampleURL, bytes.NewBuffer(tempJSON))
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", authToken)
		resp, err := netClient.Do(req)

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
