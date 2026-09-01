package client_test

import (
	"strings"
	"testing"

	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/server"
)

// TestExportSecrets checks that an export leaves the token and device key out
// unless asked, and carries both when asked.
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

	seed, pubKey, err := client.GetDeviceKey(nc)
	if err != nil {
		t.Fatal(err)
	}

	plain, err := client.ExportNodes(nc, root.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(plain), "the-token") || strings.Contains(string(plain), seed) {
		t.Fatalf("export carries secrets without -secrets:\n%s", plain)
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
	if !strings.Contains(string(full), "authToken: the-token") ||
		!strings.Contains(string(full), "deviceKey: "+seed) {
		t.Fatalf("export with -secrets is missing them:\n%s", full)
	}
	if strings.HasPrefix(string(full), "#") {
		t.Fatalf("export with -secrets should not say anything was left out:\n%s", full)
	}

	// applying a file with a device key installs it; the same file again
	// does nothing
	newSeed, newPub, err := client.GenerateDeviceKey()
	if err != nil {
		t.Fatal(err)
	}
	file := []byte("deviceKey: " + newSeed + "\n")

	plan, err := client.ImportNodes(nc, file, "test", true)
	if err != nil {
		t.Fatal(err)
	}
	if plan.DeviceKey != newPub || !strings.Contains(plan.String(), "install device key "+newPub) {
		t.Fatalf("dry run did not plan the key install: %v", plan.String())
	}
	if _, cur, _ := client.GetDeviceKey(nc); cur != pubKey {
		t.Fatal("dry run installed the key")
	}

	if _, err := client.ImportNodes(nc, file, "test", false); err != nil {
		t.Fatal(err)
	}
	if _, cur, _ := client.GetDeviceKey(nc); cur != newPub {
		t.Fatalf("key not installed: %v", cur)
	}

	plan, err = client.ImportNodes(nc, file, "test", true)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Empty() {
		t.Fatalf("second apply is not a no-op: %v", plan.String())
	}

	if _, err := client.ImportNodes(nc, []byte("deviceKey: junk\n"), "test", false); err == nil {
		t.Fatal("bad device key accepted")
	}
}
