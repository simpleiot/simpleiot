package client_test

import (
	"testing"

	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/server"
)

func TestDb(t *testing.T) {
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
		URI:         "https://localhost:8086",
		Org:         "siot-test",
		Bucket:      "test",
	}

	// hydrate database with test data
	err = client.SendNodeType(nc, dbConfig, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}
}
