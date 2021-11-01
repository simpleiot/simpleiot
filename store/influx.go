package store

import (
	"errors"
	"log"
	"strconv"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	api "github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/simpleiot/simpleiot/data"
)

// InfluxConfig represents an influxdb config
type InfluxConfig struct {
	URL    string
	Token  string
	Org    string
	Bucket string
}

// NodeToInfluxConfig converts a node to an influx config
func NodeToInfluxConfig(node data.NodeEdge) (*InfluxConfig, error) {
	ret := &InfluxConfig{}
	var ok bool
	ret.Token, ok = node.Points.Text(data.PointTypeAuthToken, "", 0)
	if !ok || ret.Token == "" {
		return ret, errors.New("Auth token must be set for InfluxDb")
	}

	ret.URL, ok = node.Points.Text(data.PointTypeURI, "", 0)
	if !ok || ret.URL == "" {
		return ret, errors.New("URL must be set for InfluxDb")
	}

	ret.Bucket, ok = node.Points.Text(data.PointTypeBucket, "", 0)
	if !ok || ret.Bucket == "" {
		return ret, errors.New("Bucket must be set for InfluxDb")
	}

	ret.Org, ok = node.Points.Text(data.PointTypeOrg, "", 0)
	if !ok || ret.Org == "" {
		return ret, errors.New("Org must be set for InfluxDb")
	}

	return ret, nil
}

// Influx represents and influxdb that we can write points to
type Influx struct {
	lastChecked time.Time
	config      InfluxConfig
	client      influxdb2.Client
	writeAPI    api.WriteAPI
	queryAPI    api.QueryAPI
}

// NewInflux creates an influx helper client
func NewInflux(config *InfluxConfig) *Influx {
	ret := &Influx{}
	ret.CheckConfig(config)
	return ret
}

// CheckConfig checks influx config and re-init if necessary
func (i *Influx) CheckConfig(config *InfluxConfig) {
	if i.config != *config {
		log.Println("Setting up new influxdb client: ", config)
		if i.client != nil {
			i.client.Close()
			i.client = nil
		}

		i.client = influxdb2.NewClient(config.URL, config.Token)
		i.writeAPI = i.client.WriteAPI(config.Org, config.Bucket)
		i.queryAPI = i.client.QueryAPI(config.Org)
		i.config = *config
	}

	i.lastChecked = time.Now()
}

// WritePoints to influxdb
func (i *Influx) WritePoints(nodeID, nodeDesc string, points data.Points) error {
	for _, point := range points {
		p := influxdb2.NewPoint("points",
			map[string]string{
				"nodeID":   nodeID,
				"nodeDesc": nodeDesc,
				"key":      point.Key,
				"type":     point.Type,
				"index":    strconv.Itoa(point.Index),
			},
			map[string]interface{}{
				"value": point.Value,
				"text":  point.Text,
			},
			point.Time)
		i.writeAPI.WritePoint(p)
	}
	return nil
}

// Close influx client
func (i *Influx) Close() {
	i.client.Close()
}
