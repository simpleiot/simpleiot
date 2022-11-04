package client_test

import (
	"testing"
	"time"

	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/server"
)

func TestAdminDbVerify(t *testing.T) {
	nc, _, stop, err := server.TestServer()

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	// give store time to init
	time.Sleep(time.Millisecond * 100)

	err = client.AdminDbVerify(nc)
	if err != nil {
		t.Fatal("Verify failed: ", err)
	}
}
