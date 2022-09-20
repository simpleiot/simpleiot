package client

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// Db represents the configuration for a SIOT DB client
type Db struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	URI         string `point:"uri"`
	Org         string `point:"org"`
	Bucket      string `point:"bucket"`
	AuthToken   string `point:"authToken"`
}

// DbClient is a SIOT database client
type DbClient struct {
	nc            *nats.Conn
	config        Db
	stop          chan struct{}
	newPoints     chan NewPoints
	newEdgePoints chan NewPoints
	newDbPoints   chan NewPoints
	upSub         *nats.Subscription
	client        influxdb2.Client
	writeAPI      api.WriteAPI
}

// NewDbClient ...
func NewDbClient(nc *nats.Conn, config Db) Client {
	// you can set things like retries, batching, precision, etc in client options.
	client := influxdb2.NewClientWithOptions(config.URI, config.AuthToken, influxdb2.DefaultOptions())
	writeAPI := client.WriteAPI(config.Org, config.Bucket)

	influxErrors := writeAPI.Errors()

	go func() {
		for {
			select {
			case err, ok := <-influxErrors:
				if err != nil {
					log.Println("Influx write error: ", err)
				}

				if !ok {
					log.Println("Influxdb write api closed")
					return
				}
			}
		}
	}()

	return &DbClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan NewPoints),
		newEdgePoints: make(chan NewPoints),
		newDbPoints:   make(chan NewPoints),
		client:        client,
		writeAPI:      writeAPI,
	}
}

// Start runs the main logic for this client and blocks until stopped
func (dbc *DbClient) Start() error {
	log.Println("Starting db client: ", dbc.config.Description)

	// FIXME, we probably want to store edge points too ...
	subject := fmt.Sprintf("up.%v.*.points", dbc.config.Parent)

	var err error
	dbc.upSub, err = dbc.nc.Subscribe(subject, func(msg *nats.Msg) {
		points, err := data.PbDecodePoints(msg.Data)
		if err != nil {
			log.Println("Error decoding points in db upSub: ", err)
			return
		}

		// find node ID for points
		chunks := strings.Split(msg.Subject, ".")
		if len(chunks) != 4 {
			log.Println("rule client up sub, malformed subject: ", msg.Subject)
			return
		}

		dbc.newDbPoints <- NewPoints{chunks[2], "", points}
	})

	if err != nil {
		return fmt.Errorf("Rule error subscribing to upsub: %v", err)
	}

	for {
		select {
		case <-dbc.stop:
			log.Println("Stopping db client: ", dbc.config.Description)
			return nil
		case pts := <-dbc.newPoints:
			err := data.MergePoints(pts.ID, pts.Points, &dbc.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}

		case pts := <-dbc.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &dbc.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}
		case pts := <-dbc.newDbPoints:
			for _, point := range pts.Points {
				p := influxdb2.NewPoint("points",
					map[string]string{
						"nodeID": pts.ID,
						"key":    point.Key,
						"type":   point.Type,
						"index":  strconv.FormatFloat(point.Index, 'f', -1, 64),
					},
					map[string]interface{}{
						"value": point.Value,
						"text":  point.Text,
					},
					point.Time)
				dbc.writeAPI.WritePoint(p)
			}
		}
	}
}

// Stop sends a signal to the Start function to exit
func (dbc *DbClient) Stop(err error) {
	close(dbc.stop)
}

// Points is called by the Manager when new points for this
// node are received.
func (dbc *DbClient) Points(nodeID string, points []data.Point) {
	dbc.newPoints <- NewPoints{nodeID, "", points}
}

// EdgePoints is called by the Manager when new edge points for this
// node are received.
func (dbc *DbClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	dbc.newEdgePoints <- NewPoints{nodeID, parentID, points}
}
