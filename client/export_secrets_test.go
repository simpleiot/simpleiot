package client_test

import (
	"strings"
	"testing"

	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/server"
)

// TestExportSecrets checks that an export leaves the token out unless asked.
func TestExportSecrets(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}
	defer stop()

	sync := client.Sync{
		ID: "sync-1", Parent: root.ID, Description: "cloud",
		URI: "nats://cloud:4222", AuthToken: "the-token",
	}
	if err := client.SendNodeType(nc, sync, "test"); err != nil {
		t.Fatal(err)
	}

	plain, err := client.ExportNodes(nc, root.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(plain), "the-token") {
		t.Fatalf("export carries the token without -secrets:\n%s", plain)
	}
	if !strings.HasPrefix(string(plain), "# authToken") {
		t.Fatalf("export does not say what it left out:\n%s", plain)
	}
	if !strings.Contains(string(plain), "uri: nats://cloud:4222") {
		t.Fatalf("export lost the rest of the sync node:\n%s", plain)
	}

	full, err := client.ExportNodes(nc, root.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(full), "authToken: the-token") {
		t.Fatalf("export with -secrets is missing the token:\n%s", full)
	}
	if strings.HasPrefix(string(full), "#") {
		t.Fatalf("export with -secrets should not say anything was left out:\n%s", full)
	}
}
