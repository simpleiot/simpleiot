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
	upSubHr       *nats.Subscription
	client        influxdb2.Client
	writeAPI      api.WriteAPI
}

// NewDbClient ...
func NewDbClient(nc *nats.Conn, config Db) Client {
	return &DbClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan NewPoints),
		newEdgePoints: make(chan NewPoints),
		newDbPoints:   make(chan NewPoints),
	}
}

// Run runs the main logic for this client and blocks until stopped
func (dbc *DbClient) Run() error {
	log.Println("Starting db client: ", dbc.config.Description)

	// FIXME, we probably want to store edge points too ...
	subject := fmt.Sprintf("up.%v.*", dbc.config.Parent)

	var err error
	dbc.upSub, err = dbc.nc.Subscribe(subject, func(msg *nats.Msg) {
		points, err := data.PbDecodePoints(msg.Data)
		if err != nil {
			log.Println("Error decoding points in db upSub: ", err)
			return
		}

		// find node ID for points
		chunks := strings.Split(msg.Subject, ".")
		if len(chunks) != 3 {
			log.Println("rule client up sub, malformed subject: ", msg.Subject)
			return
		}

		dbc.newDbPoints <- NewPoints{chunks[2], "", points}
	})

	if err != nil {
		return err
	}

	subjectHR := fmt.Sprintf("phrup.%v.*", dbc.config.Parent)

	dbc.upSubHr, err = dbc.nc.Subscribe(subjectHR, func(msg *nats.Msg) {
		points, err := data.PbDecodePoints(msg.Data)
		if err != nil {
			log.Println("Error decoding points in db upSubHr: ", err)
			return
		}

		// find node ID for points
		chunks := strings.Split(msg.Subject, ".")
		if len(chunks) != 3 {
			log.Println("rule client up hr sub, malformed subject: ", msg.Subject)
			return
		}

		dbc.newDbPoints <- NewPoints{chunks[2], "", points}
	})

	if err != nil {
		return fmt.Errorf("Rule error subscribing to upsub: %v", err)
	}

	setupAPI := func() {
		log.Println("Setting up Influx API")
		// you can set things like retries, batching, precision, etc in client options.
		dbc.client = influxdb2.NewClientWithOptions(dbc.config.URI,
			dbc.config.AuthToken, influxdb2.DefaultOptions())
		dbc.writeAPI = dbc.client.WriteAPI(dbc.config.Org, dbc.config.Bucket)

		influxErrors := dbc.writeAPI.Errors()

		go func() {
			for err := range influxErrors {
				if err != nil {
					log.Println("Influx write error: ", err)
				}

			}
			log.Println("Influxdb write api closed")
		}()
	}

	setupAPI()

done:
	for {
		select {
		case <-dbc.stop:
			log.Println("Stopping db client: ", dbc.config.Description)
			break done
		case pts := <-dbc.newPoints:
			err := data.MergePoints(pts.ID, pts.Points, &dbc.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}

			for _, p := range pts.Points {
				switch p.Type {
				case data.PointTypeURI,
					data.PointTypeOrg,
					data.PointTypeBucket,
					data.PointTypeAuthToken:
					// we need to restart the influx write API
					dbc.client.Close()
					setupAPI()
				}
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
						"index":  strconv.FormatFloat(float64(point.Index), 'f', -1, 64),
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

	// clean up
	dbc.client.Close()
	return nil
}

// Stop sends a signal to the Run function to exit
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
