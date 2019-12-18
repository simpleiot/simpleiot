package db

import (
	"github.com/cbrake/influxdbhelper/v2"
	client "github.com/influxdata/influxdb1-client/v2"
	"github.com/simpleiot/simpleiot/data"
)

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
func (i *Influx) WriteSamples(samples []data.Sample) error {
	for _, s := range samples {
		err := i.client.UseMeasurement("samples").WritePoint(s)
		if err != nil {
			return err
		}
	}

	return nil
}
