package client_test

import (
	"testing"
	"time"

	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

// TestRules populates a rule in the system that watches
// a variable and when set, sets another variable. This
// tests out the basic rule logic.
func TestRules(t *testing.T) {
	nc, root, stop, err := server.TestServer()

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	// send test nodes to Db
	vin := client.Variable{
		ID:          "ID-varin",
		Parent:      root.ID,
		Description: "var in",
	}

	err = client.SendNodeType(nc, vin, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	vout := client.Variable{
		ID:          "ID-varout",
		Parent:      root.ID,
		Description: "var out",
	}

	err = client.SendNodeType(nc, vout, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	r := client.Rule{
		ID:          "ID-rule",
		Parent:      root.ID,
		Description: "test rule",
		Disabled:    false,
	}

	err = client.SendNodeType(nc, r, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	c := client.Condition{
		ID:            "ID-condition",
		Parent:        r.ID,
		Description:   "cond vin high",
		ConditionType: data.PointValuePointValue,
		PointType:     data.PointTypeValue,
		ValueType:     data.PointValueOnOff,
		NodeID:        vin.ID,
		Operator:      data.PointValueEqual,
		Value:         1,
	}

	err = client.SendNodeType(nc, c, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	a := client.Action{
		ID:          "ID-action-active",
		Parent:      r.ID,
		Description: "action active",
		Action:      data.PointValueSetValue,
		PointType:   data.PointTypeValue,
		NodeID:      vout.ID,
		Value:       1,
	}

	err = client.SendNodeType(nc, a, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	// FIXME:
	// this delay is required to work around a bug in the manager
	// where it is resetting and does not see the ActionInactive points
	// See https://github.com/simpleiot/simpleiot/issues/630
	// the tools/test-rules.sh script can be used to test a fix for this
	// problem
	time.Sleep(100 * time.Millisecond)

	a2 := client.ActionInactive{
		ID:          "ID-action-inactive",
		Parent:      r.ID,
		Description: "action inactive",
		Action:      data.PointValueSetValue,
		PointType:   data.PointTypeValue,
		NodeID:      vout.ID,
		Value:       0,
	}

	err = client.SendNodeType(nc, a2, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	// set up a node watcher to watch the output variable
	voutGet, voutStop, err := client.NodeWatcher[client.Variable](nc, vout.ID, vout.Parent)

	if err != nil {
		t.Fatal("Error setting up watcher")
	}

	defer voutStop()

	if voutGet().Value != 0 {
		t.Fatal("initial vout value is not correct")
	}

	// wait for rule to get set up
	time.Sleep(250 * time.Millisecond)

	// set vin and look for vout to change
	err = client.SendNodePoint(nc, vin.ID, data.Point{Type: data.PointTypeValue,
		Value: 1, Origin: "test"}, true)

	if err != nil {
		t.Errorf("Error sending point: %v", err)
	}

	start := time.Now()
	for {
		if voutGet().Value == 1 {
			// all is well
			break
		}
		if time.Since(start) > time.Second {
			t.Fatal("Timeout waiting for vout to be set")
		}
		<-time.After(time.Millisecond * 10)
	}

	// clear vin and look for vout to change
	err = client.SendNodePoint(nc, vin.ID, data.Point{Type: data.PointTypeValue,
		Value: 0, Origin: "test"}, true)

	if err != nil {
		t.Errorf("Error sending point: %v", err)
	}

	start = time.Now()
	for {
		if voutGet().Value == 0 {
			// all is well
			break
		}
		if time.Since(start) > time.Second {
			t.Fatal("Timeout waiting for vout to be cleared")
		}
		<-time.After(time.Millisecond * 10)
	}
}

/*
leave everything enabled and toggle vin and watch vout toggle -- same as the TestRules() function. This ensures that your test is setup correctly.
disable rule, set vin and verify vout does not get set. Then clear vin.
enable rule, and disable condition. set vin and verify vout does not get set. Clear vin.
enable condition, and disable action. set vin and verify vout does not get set. Clear vin.
enable action, set vin, then disable rule. verify vout gets cleared.
enable rule, and verify vout gets set.
disable condition, and verify vout gets cleared.
enable condition, and verify vout gets set.
*/
func TestDisabled(t *testing.T) {
	nc, root, stop, err := server.TestServer()

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	// send test nodes to Db
	vin := client.Variable{
		ID:          "ID-varin",
		Parent:      root.ID,
		Description: "var in",
	}

	err = client.SendNodeType(nc, vin, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	vout := client.Variable{
		ID:          "ID-varout",
		Parent:      root.ID,
		Description: "var out",
	}

	err = client.SendNodeType(nc, vout, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	r := client.Rule{
		ID:          "ID-rule",
		Parent:      root.ID,
		Description: "test rule",
		Disabled:    false,
	}

	err = client.SendNodeType(nc, r, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	c := client.Condition{
		ID:            "ID-condition",
		Parent:        r.ID,
		Description:   "cond vin high",
		ConditionType: data.PointValuePointValue,
		PointType:     data.PointTypeValue,
		ValueType:     data.PointValueOnOff,
		NodeID:        vin.ID,
		Operator:      data.PointValueEqual,
		Value:         1,
	}

	err = client.SendNodeType(nc, c, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	a := client.Action{
		ID:          "ID-action-active",
		Parent:      r.ID,
		Description: "action active",
		Action:      data.PointValueSetValue,
		PointType:   data.PointTypeValue,
		NodeID:      vout.ID,
		Value:       1,
	}

	err = client.SendNodeType(nc, a, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	// FIXME:
	// this delay is required to work around a bug in the manager
	// where it is resetting and does not see the ActionInactive points
	// See https://github.com/simpleiot/simpleiot/issues/630
	// the tools/test-rules.sh script can be used to test a fix for this
	// problem
	time.Sleep(100 * time.Millisecond)

	a2 := client.ActionInactive{
		ID:          "ID-action-inactive",
		Parent:      r.ID,
		Description: "action inactive",
		Action:      data.PointValueSetValue,
		PointType:   data.PointTypeValue,
		NodeID:      vout.ID,
		Value:       0,
	}

	err = client.SendNodeType(nc, a2, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	// set up a node watcher to watch the output variable
	voutGet, voutStop, err := client.NodeWatcher[client.Variable](nc, vout.ID, vout.Parent)

	if err != nil {
		t.Fatal("Error setting up watcher")
	}

	defer voutStop()

	if voutGet().Value != 0 {
		t.Fatal("initial vout value is not correct")
	}

	// wait for rule to get set up
	time.Sleep(250 * time.Millisecond)

	/*
		leave everything enabled and toggle vin and watch vout toggle -- same as the TestRules() function.
		This ensures that your test is setup correctly.
	*/
	// set vin and look for vout to change
	err = client.SendNodePoint(nc, vin.ID, data.Point{Type: data.PointTypeValue,
		Value: 1, Origin: "test"}, true)

	if err != nil {
		t.Errorf("Error sending point: %v", err)
	}

	start := time.Now()
	for {
		if voutGet().Value == 1 {
			// all is well
			break
		}
		if time.Since(start) > time.Second {
			t.Fatal("Timeout waiting for vout to be set")
		}
		<-time.After(time.Millisecond * 10)
	}

	// clear vin and look for vout to change
	err = client.SendNodePoint(nc, vin.ID, data.Point{Type: data.PointTypeValue,
		Value: 0, Origin: "test"}, true)

	if err != nil {
		t.Errorf("Error sending point: %v", err)
	}

	start = time.Now()
	for {
		if voutGet().Value == 0 {
			// all is well
			break
		}
		if time.Since(start) > time.Second {
			t.Fatal("Timeout waiting for vout to be cleared")
		}
		<-time.After(time.Millisecond * 10)
	}

	/*
		disable rule, set vin and verify vout does not get set. Then clear vin.
	*/
	// disable rule
	err = client.SendNodePoint(nc, r.ID, data.Point{Type: data.PointTypeDisabled,
		Value: 1, Origin: "test"}, true)
	if err != nil {
		t.Errorf("Error sending point: %v", err)
	}
	//set vin
	err = client.SendNodePoint(nc, vin.ID, data.Point{Type: data.PointTypeValue,
		Value: 1, Origin: "test"}, true)
	if err != nil {
		t.Errorf("Error sending point: %v", err)
	}

	// verify vout does not get set
	start = time.Now()
	for {
		if voutGet().Value == 0 {
			// all is well
			break
		}
		if time.Since(start) > time.Second {
			t.Fatal("Timeout waiting for vout to be cleared")
		}
		<-time.After(time.Millisecond * 10)
	}

	//clear vin
	err = client.SendNodePoint(nc, vin.ID, data.Point{Type: data.PointTypeValue,
		Value: 0, Origin: "test"}, true)
	if err != nil {
		t.Errorf("Error sending point: %v", err)
	}

	/*
		enable rule, and disable condition. set vin and verify vout does not get set. Clear vin.
	*/
	// enable rule
	err = client.SendNodePoint(nc, r.ID, data.Point{Type: data.PointTypeDisabled,
		Value: 0, Origin: "test"}, true)
	if err != nil {
		t.Errorf("Error sending point: %v", err)
	}

	// disable condition
	err = client.SendNodePoint(nc, c.ID, data.Point{Type: data.PointTypeDisabled,
		Value: 1, Origin: "test"}, true)
	if err != nil {
		t.Errorf("Error sending point: %v", err)
	}

	//set vin
	err = client.SendNodePoint(nc, vin.ID, data.Point{Type: data.PointTypeValue,
		Value: 1, Origin: "test"}, true)
	if err != nil {
		t.Errorf("Error sending point: %v", err)
	}

	// verify vout does not get set
	start = time.Now()
	for {
		if voutGet().Value == 0 {
			// all is well
			break
		}
		if time.Since(start) > time.Second {
			t.Fatal("Timeout waiting for vout to be cleared")
		}
		<-time.After(time.Millisecond * 10)
	}

	//clear vin
	err = client.SendNodePoint(nc, vin.ID, data.Point{Type: data.PointTypeValue,
		Value: 0, Origin: "test"}, true)
	if err != nil {
		t.Errorf("Error sending point: %v", err)
	}

	/*
		enable condition, and disable action. set vin and verify vout does not get set. Clear vin.
	*/

	// enable condition
	err = client.SendNodePoint(nc, c.ID, data.Point{Type: data.PointTypeDisabled,
		Value: 0, Origin: "test"}, true)
	if err != nil {
		t.Errorf("Error sending point: %v", err)
	}

	//disable action
	err = client.SendNodePoint(nc, a.ID, data.Point{Type: data.PointTypeDisabled,
		Value: 1, Origin: "test"}, true)
	if err != nil {
		t.Errorf("Error sending point: %v", err)
	}

	//set vin
	err = client.SendNodePoint(nc, vin.ID, data.Point{Type: data.PointTypeValue,
		Value: 1, Origin: "test"}, true)
	if err != nil {
		t.Errorf("Error sending point: %v", err)
	}

	// verify vout does not get set
	start = time.Now()
	for {
		if voutGet().Value == 0 {
			// all is well
			break
		}
		if time.Since(start) > time.Second {
			t.Fatal("Timeout waiting for vout to be cleared")
		}
		<-time.After(time.Millisecond * 10)
	}

	//clear vin
	err = client.SendNodePoint(nc, vin.ID, data.Point{Type: data.PointTypeValue,
		Value: 0, Origin: "test"}, true)
	if err != nil {
		t.Errorf("Error sending point: %v", err)
	}

	/*
		enable action, set vin, then disable rule. verify vout gets cleared.
	*/

	//enable action
	err = client.SendNodePoint(nc, a.ID, data.Point{Type: data.PointTypeDisabled,
		Value: 0, Origin: "test"}, true)
	if err != nil {
		t.Errorf("Error sending point: %v", err)
	}

	//set vin
	err = client.SendNodePoint(nc, vin.ID, data.Point{Type: data.PointTypeValue,
		Value: 1, Origin: "test"}, true)
	if err != nil {
		t.Errorf("Error sending point: %v", err)
	}

	//disable rule
	err = client.SendNodePoint(nc, r.ID, data.Point{Type: data.PointTypeDisabled,
		Value: 1, Origin: "test"}, true)
	if err != nil {
		t.Errorf("Error sending point: %v", err)
	}

	// verify vout gets cleared.
	start = time.Now()
	for {
		if voutGet().Value == 0 {
			// all is well
			break
		}
		if time.Since(start) > time.Second {
			t.Fatal("Timeout waiting for vout to be cleared")
		}
		<-time.After(time.Millisecond * 10)
	}

	/*
		enable rule, and verify vout gets set.
	*/

	//enable rule
	err = client.SendNodePoint(nc, r.ID, data.Point{Type: data.PointTypeDisabled,
		Value: 0, Origin: "test"}, true)
	if err != nil {
		t.Errorf("Error sending point: %v", err)
	}

	// verify vout gets set
	start = time.Now()
	for {
		if voutGet().Value == 1 {
			// all is well
			break
		}
		if time.Since(start) > time.Second {
			t.Fatal("Timeout waiting for vout to be cleared")
		}
		<-time.After(time.Millisecond * 10)
	}

	/*
		disable condition, and verify vout gets cleared.
	*/

	//disable condition
	err = client.SendNodePoint(nc, c.ID, data.Point{Type: data.PointTypeDisabled,
		Value: 1, Origin: "test"}, true)
	if err != nil {
		t.Errorf("Error sending point: %v", err)
	}

	// verify vout gets cleared.
	start = time.Now()
	for {
		if voutGet().Value == 0 {
			// all is well
			break
		}
		if time.Since(start) > time.Second {
			t.Fatal("Timeout waiting for vout to be cleared")
		}
		<-time.After(time.Millisecond * 10)
	}

	/*
		enable condition, and verify vout gets set.
	*/

	//enable condition
	err = client.SendNodePoint(nc, c.ID, data.Point{Type: data.PointTypeDisabled,
		Value: 0, Origin: "test"}, true)
	if err != nil {
		t.Errorf("Error sending point: %v", err)
	}

	// verify vout gets set.
	start = time.Now()
	for {
		if voutGet().Value == 1 {
			// all is well
			break
		}
		if time.Since(start) > time.Second {
			t.Fatal("Timeout waiting for vout to be cleared")
		}
		<-time.After(time.Millisecond * 10)
	}
}

/*
if one condition is active and the 2nd condition is disabled, the rule fires
if both conditions are disabled, the rule is inactive.
*/
func TestDisabledCondition(t *testing.T) {
	nc, root, stop, err := server.TestServer()

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	// send test nodes to Db
	vin := client.Variable{
		ID:          "ID-varin",
		Parent:      root.ID,
		Description: "var in",
	}

	err = client.SendNodeType(nc, vin, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	vout := client.Variable{
		ID:          "ID-varout",
		Parent:      root.ID,
		Description: "var out",
	}

	err = client.SendNodeType(nc, vout, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	r := client.Rule{
		ID:          "ID-rule",
		Parent:      root.ID,
		Description: "test rule",
		Disabled:    false,
	}

	err = client.SendNodeType(nc, r, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	c := client.Condition{
		ID:            "ID-condition",
		Parent:        r.ID,
		Description:   "cond vin high",
		ConditionType: data.PointValuePointValue,
		PointType:     data.PointTypeValue,
		ValueType:     data.PointValueOnOff,
		NodeID:        vin.ID,
		Operator:      data.PointValueEqual,
		Value:         1,
	}

	err = client.SendNodeType(nc, c, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	time.Sleep(100 * time.Millisecond)

	c2 := client.Condition{
		ID:            "ID-disabled-condition",
		Parent:        r.ID,
		Description:   "cond vin high",
		ConditionType: data.PointValuePointValue,
		PointType:     data.PointTypeValue,
		ValueType:     data.PointValueOnOff,
		NodeID:        vin.ID,
		Operator:      data.PointValueEqual,
		Value:         0,
		Disabled:      true,
	}

	err = client.SendNodeType(nc, c2, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	a := client.Action{
		ID:          "ID-action-active",
		Parent:      r.ID,
		Description: "action active",
		Action:      data.PointValueSetValue,
		PointType:   data.PointTypeValue,
		NodeID:      vout.ID,
		Value:       1,
	}

	err = client.SendNodeType(nc, a, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	// FIXME:
	// this delay is required to work around a bug in the manager
	// where it is resetting and does not see the ActionInactive points
	// See https://github.com/simpleiot/simpleiot/issues/630
	// the tools/test-rules.sh script can be used to test a fix for this
	// problem
	time.Sleep(100 * time.Millisecond)

	a2 := client.ActionInactive{
		ID:          "ID-action-inactive",
		Parent:      r.ID,
		Description: "action inactive",
		Action:      data.PointValueSetValue,
		PointType:   data.PointTypeValue,
		NodeID:      vout.ID,
		Value:       0,
	}

	err = client.SendNodeType(nc, a2, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	// set up a node watcher to watch the output variable
	voutGet, voutStop, err := client.NodeWatcher[client.Variable](nc, vout.ID, vout.Parent)

	if err != nil {
		t.Fatal("Error setting up watcher")
	}

	defer voutStop()

	if voutGet().Value != 0 {
		t.Fatal("initial vout value is not correct")
	}

	// wait for rule to get set up
	time.Sleep(250 * time.Millisecond)

	/*
		if one condition is active and the 2nd condition is disabled, the rule fires
	*/

	// set vin and look for vout to change

	err = client.SendNodePoint(nc, vin.ID, data.Point{Type: data.PointTypeValue,
		Value: 1, Origin: "test"}, true)

	if err != nil {
		t.Errorf("Error sending point: %v", err)
	}

	start := time.Now()
	for {
		if voutGet().Value == 1 {
			// all is well
			break
		}
		if time.Since(start) > time.Second {
			t.Fatal("Timeout waiting for vout to be set")
		}
		<-time.After(time.Millisecond * 10)
	}

	// clear vin
	err = client.SendNodePoint(nc, vin.ID, data.Point{Type: data.PointTypeValue,
		Value: 0, Origin: "test"}, true)

	if err != nil {
		t.Errorf("Error sending point: %v", err)
	}

	/*
		if both conditions are disabled, the rule is inactive.
	*/
	err = client.SendNodePoint(nc, c.ID, data.Point{Type: data.PointTypeDisabled,
		Value: 1, Origin: "test"}, true)
	if err != nil {
		t.Errorf("Error sending point: %v", err)
	}

	err = client.SendNodePoint(nc, vin.ID, data.Point{Type: data.PointTypeValue,
		Value: 1, Origin: "test"}, true)
	if err != nil {
		t.Errorf("Error sending point: %v", err)
	}

	start = time.Now()
	for {
		if voutGet().Value == 0 {
			// all is well
			break
		}
		if time.Since(start) > time.Second {
			t.Fatal("Timeout waiting for vout to be cleared")
		}
		<-time.After(time.Millisecond * 10)
	}
}
