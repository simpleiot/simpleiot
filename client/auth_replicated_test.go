package client_test

import (
	"testing"
	"time"

	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

// TestReplicatedUserSignIn: a user created on a device is replicated
// upstream with the rest of the device's tree, and signs in on the device
// only. A device's default admin account is not a way into the upstream.
func TestReplicatedUserSignIn(t *testing.T) {
	ncU, rootU, optsU, stopU := credUpstream(t)
	defer stopU()

	ncD, rootD, stopD, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting downstream test server: ", err)
	}
	defer stopD()

	enrollDevice(t, ncU, rootU, rootD.ID, "cred-1", devicePubKey(t, ncD))
	startDeviceSync(t, ncD, rootD, optsU.NatsServer)
	waitFor(t, 15*time.Second, "credential not marked connected", func() bool {
		return getCred(t, ncU, "cred-1").Connected
	})

	// change the upstream's own admin password, so admin/admin only
	// exists on the device
	admins, err := client.GetNodesType[client.User](ncU, rootU.ID, "all")
	if err != nil || len(admins) == 0 {
		t.Fatalf("upstream admin: %v %v", admins, err)
	}
	err = client.SendNodePoint(ncU, admins[0].ID,
		data.NewPointString(data.PointTypePass, "", "upstream-pw"), true)
	if err != nil {
		t.Fatal(err)
	}

	// a user on the device
	u := client.User{ID: "dev-user", Parent: rootD.ID, FirstName: "d",
		Email: "d@example.com", Pass: "dev-pw"}
	if err := client.SendNodeType(ncD, u, "test"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, 10*time.Second, "user not replicated upstream", func() bool {
		nodes, err := client.GetNodes(ncU, "all", "dev-user", "", false)
		if err != nil || len(nodes) == 0 {
			return false
		}
		pass, _ := nodes[0].Points.Text(data.PointTypePass, "")
		return pass != ""
	})

	// signs in on the device
	nodes, err := client.UserCheck(ncD, "d@example.com", "dev-pw")
	if err != nil || len(nodes) == 0 {
		t.Fatalf("device sign-in: %v %v", nodes, err)
	}

	// not on the upstream, and neither does the device's admin
	for _, tt := range []struct{ email, pass string }{
		{"d@example.com", "dev-pw"},
		{"admin", "admin"},
	} {
		nodes, err := client.UserCheck(ncU, tt.email, tt.pass)
		if err == nil && len(nodes) > 0 {
			t.Fatalf("%v signed in on the upstream: %v", tt.email, nodes)
		}
	}

	// the upstream's own admin still does
	nodes, err = client.UserCheck(ncU, "admin", "upstream-pw")
	if err != nil || len(nodes) == 0 {
		t.Fatalf("upstream admin sign-in: %v %v", nodes, err)
	}

	// setting the device user's password from the upstream administers
	// the device: the new password works on the device and the user is
	// still not an account on the upstream
	err = client.SendNodePoint(ncU, "dev-user",
		data.NewPointString(data.PointTypePass, "", "set-upstream"), true)
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, 10*time.Second, "password change not on device", func() bool {
		nodes, err := client.UserCheck(ncD, "d@example.com", "set-upstream")
		return err == nil && len(nodes) > 0
	})

	nodes, err = client.UserCheck(ncU, "d@example.com", "set-upstream")
	if err == nil && len(nodes) > 0 {
		t.Fatalf("device user signed in on the upstream after an upstream password change: %v", nodes)
	}
}
