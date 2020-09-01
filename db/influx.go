package db

import (
	"time"

	"github.com/cbrake/influxdbhelper/v2"
	client "github.com/influxdata/influxdb1-client/v2"
	"github.com/simpleiot/simpleiot/data"
)

// InfluxSample represents a sample that is written into influxdb
type InfluxSample struct {
	// Type of sample (voltage, current, key, etc)
	Type string `influx:"type,tag"`

	// ID of the sensor that provided the sample
	ID string `influx:"id,tag"`

	// DeviceID of the ID of the device that provided the sample
	DeviceID string `influx:"deviceId,tag"`

	// Average OR
	// Instantaneous analog or digital value of the sample.
	// 0 and 1 are used to represent digital values
	Value float64 `influx:"value"`

	// statistical values that may be calculated
	Min float64 `influx:"min"`
	Max float64 `influx:"max"`

	// Time the sample was taken
	Time time.Time `influx:"time"`

	// Duration over which the sample was taken
	Duration time.Duration `influx:"duration"`
}

// PointToInfluxSample converts a sample to influx sample
func PointToInfluxSample(deviceID string, p data.Point) InfluxSample {
	return InfluxSample{
		Type:     p.Type,
		ID:       p.ID,
		DeviceID: deviceID,
		Value:    p.Value,
		Min:      p.Min,
		Max:      p.Max,
		Time:     p.Time,
		Duration: p.Duration,
	}
}

// Influx represents and influxdb that we can write samples to
type Influx struct {
	client influxdbhelper.Client
}

// NewInflux creates an influx helper client
func NewInflux(url, dbName, user, password string) (*Influx, error) {
	c, err := influxdbhelper.NewClient(url, user, password, "ns")
	if err != nil {
		return nil, err
	}

	c = c.UseDB(dbName)

	// Create test database if it doesn't already exist
	q := client.NewQuery("CREATE DATABASE "+dbName, "", "")
	res, err := c.Query(q)
	if err != nil {
		return nil, err
	}
	if res.Error() != nil {
		return nil, res.Error()
	}

	return &Influx{
		client: c,
	}, nil
}

// WriteSamples to influxdb
func (i *Influx) WriteSamples(samples []InfluxSample) error {
	for _, s := range samples {
		err := i.client.UseMeasurement("samples").WritePoint(s)
		if err != nil {
			return err
		}
	}

	return nil
}
