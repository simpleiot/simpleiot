package client_test

import (
	"testing"
	"time"

	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

func TestAuthDefault(t *testing.T) {
	nc, _, stop, err := server.TestServer()

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	ne, err := client.UserCheck(nc, "admin", "admin")
	if err != nil {
		t.Fatal("User check error: ", err)
	}

	if len(ne) < 2 {
		t.Fatal("Expected at least two nodes from auth request")
	}

	// the default admin user is created with a legacy plaintext password,
	// so a successful login must rewrite it as a bcrypt hash
	adminID := ne[0].ID
	rehashed := false
	for range 50 {
		nodes, err := client.GetNodes(nc, "all", adminID, "", false)
		if err != nil {
			t.Fatal("Error getting admin node: ", err)
		}
		if len(nodes) > 0 {
			pass, _ := nodes[0].Points.Text(data.PointTypePass, "")
			if data.PasswordIsHashed(pass) {
				rehashed = true
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}

	if !rehashed {
		t.Fatal("legacy plaintext password was not rehashed after login")
	}

	// the rehashed password must still authenticate
	ne, err = client.UserCheck(nc, "admin", "admin")
	if err != nil {
		t.Fatal("User check error after rehash: ", err)
	}

	if len(ne) < 2 {
		t.Fatal("Expected at least two nodes from auth request after rehash")
	}
}

func TestAuthPasswordHashing(t *testing.T) {
	nc, root, stop, err := server.TestServer()

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	u := client.User{ID: "hash-user", Parent: root.ID,
		Email: "hash@example.com", Pass: "mypass"}

	err = client.SendNodeType(nc, u, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	// the store hashes the pass point on write, so the stored value must
	// be a bcrypt hash, not the plaintext password
	nodes, err := client.GetNodes(nc, "all", "hash-user", "", false)
	if err != nil {
		t.Fatal("Error getting user node: ", err)
	}
	if len(nodes) < 1 {
		t.Fatal("user node not found")
	}

	pass, _ := nodes[0].Points.Text(data.PointTypePass, "")
	if !data.PasswordIsHashed(pass) {
		t.Fatal("stored password is not hashed: ", pass)
	}
	if pass == "mypass" {
		t.Fatal("stored password is plaintext")
	}

	// correct password authenticates
	ne, err := client.UserCheck(nc, "hash@example.com", "mypass")
	if err != nil {
		t.Fatal("User check error: ", err)
	}
	if len(ne) < 2 {
		t.Fatal("Expected at least two nodes from auth request")
	}

	// wrong password does not
	ne, err = client.UserCheck(nc, "hash@example.com", "wrong")
	if err != nil {
		t.Fatal("User check error: ", err)
	}
	if len(ne) != 0 {
		t.Fatal("wrong password should not authenticate")
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
