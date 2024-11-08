package client_test

import (
	"testing"

	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/server"
)

func TestAuthDefault(t *testing.T) {
	nc, _, stop, err := server.TestServer()

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	ne, err := client.UserCheck(nc, "admin@admin.com", "admin")
	if err != nil {
		t.Fatal("User check error: ", err)
	}

	if len(ne) < 2 {
		t.Fatal("Expected at least two nodes from auth request")
	}
}

func TestAuthMovedUser(t *testing.T) {
	nc, root, stop, err := server.TestServer()

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	u := client.User{ID: "test-user", Parent: root.ID,
		Email: "test", Pass: "test"}

	err = client.SendNodeType(nc, u, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	g := client.Group{ID: "test-group", Parent: root.ID, Description: "testg"}
	err = client.SendNodeType(nc, g, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	// verify new user auths in current location
	ne, err := client.UserCheck(nc, "test", "test")
	if err != nil {
		t.Fatal("User check error: ", err)
	}

	if len(ne) < 2 {
		t.Fatal("Expected at least two nodes from auth request")
	}

	// move user to group and try again
	err = client.MoveNode(nc, "test-user", root.ID, "test-group", "test")
	if err != nil {
		t.Fatal("Error moving node: ", err)
	}

	ne, err = client.UserCheck(nc, "test", "test")
	if err != nil {
		t.Fatal("User check error: ", err)
	}

	if len(ne) < 2 {
		t.Fatal("after move, expected at least two nodes from auth request: ", len(ne))
	}
}
