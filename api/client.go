package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/simpleiot/simpleiot/data"
)

// Point is a custom value of data.Point with Time set to a pointer. This allows
// omitempty to work for zero timestamps to avoid bloating JSON packets.
type Point struct {
	// Type of point (voltage, current, key, etc)
	Type string `json:"type,omitempty" influx:"type,tag"`

	// Key of the device that provided the point
	Key string `json:"key,omitempty" influx:"key,tag"`

	// Average OR
	// Instantaneous analog or digital value of the point.
	// 0 and 1 are used to represent digital values
	Value float64 `json:"value,omitempty" influx:"value"`

	// Time the point was taken
	Time *time.Time `json:"time,omitempty" gob:"-" influx:"time"`

	// Duration over which the point was taken
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
		Key:   s.Key,
		Value: s.Value,
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
		pointURL := portalURL + "/v1/devices/" + deviceID + "/points"

		tempJSON, err := json.Marshal(NewPoints(points))
		if err != nil {
			return err
		}

		if debug {
			log.Println("Sending points:", string(tempJSON))
		}

		req, err := http.NewRequest("POST", pointURL, bytes.NewBuffer(tempJSON))
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
			errstring := "Server error: " + resp.Status + " " + pointURL
			body, _ := io.ReadAll(resp.Body)
			errstring += " " + string(body)
			return errors.New(errstring)
		}

		return nil
	}
}
