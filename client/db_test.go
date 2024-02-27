package client_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"

	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

func checkPort(host string, port string) error {
	timeout := time.Second
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
	if err != nil {
		return fmt.Errorf("Connecting error: %v", err)
	}
	if conn != nil {
		defer conn.Close()
	}

	return nil
}

func TestDb(t *testing.T) {
	// check if there is an influxdb server running, IE skip this test in CI runs
	err := checkPort("localhost", "8086")
	if err != nil {
		fmt.Println("Error opening influx port, skipping TestDb: ", err)
		t.Skip("Error opening Influx port")
	}

	authToken := os.Getenv("INFLUX_AUTH_TOKEN")
	if authToken == "" {
		t.Skip("Environment variable INFLUX_AUTH_TOKEN is not set")
	}

	// Start up a SIOT test server for this test
	nc, root, stop, err := server.TestServer()
	_ = nc

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	dbConfig := client.Db{
		ID:          "ID-db",
		Parent:      root.ID,
		Description: "influxdb",
		URI:         "http://localhost:8086",
		Org:         "siot-test",
		Bucket:      "test",
		AuthToken:   authToken,
	}

	// set up Db client
	err = client.SendNodeType(nc, dbConfig, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	// connect to influx
	iClient := influxdb2.NewClient(dbConfig.URI, dbConfig.AuthToken)
	iQuery := iClient.QueryAPI(dbConfig.Org)
	_ = iQuery

	// wait for client to start
	time.Sleep(time.Millisecond * 100)

	// write a point and then see if it shows up in influxdb
	err = client.SendNodePoint(nc, dbConfig.ID,
		data.Point{Type: data.PointTypeDescription, Text: "updated description", Origin: "test"}, true)

	if err != nil {
		t.Fatal("Error sending points")
	}

	query := fmt.Sprintf(`
		from(bucket:"test")
		  |> range(start: -15m)
		  |> filter(fn: (r) => r._measurement == "points" and r.nodeID == "%v" and
		  	r.type == "description" and r._field == "text")
		  |> last()
	`, dbConfig.ID)

	// points are batched by the influx client and can take up to 1s to be written
	time.Sleep(time.Second * 1)

	result, err := iQuery.Query(context.Background(), query)
	if err != nil {
		t.Fatal("influx query failed: ", err)
	}

	var pTime time.Time
	var pValue string

	for result.Next() {
		r := result.Record()
		pTime = r.Time()
		pValue = r.Value().(string)
	}

	err = result.Err()

	if err != nil {
		t.Fatal("influx result error: ", err)
	}

	if time.Since(pTime) > time.Second*4 {
		t.Fatal("Did not get a point recently: ", time.Since(pTime), pTime)
	}

	if pValue != "updated description" {
		t.Fatal("Point value not correct")
	}
}
