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

	if voutGet().Value["0"] != 0 {
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
		if voutGet().Value["0"] == 1 {
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
		if voutGet().Value["0"] == 0 {
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

	if voutGet().Value["0"] != 0 {
		t.Fatal("initial vout value is not correct")
	}

	// wait for rule to get set up
	time.Sleep(250 * time.Millisecond)

	lastvout := float64(0)

	// set change to true if you are execting vout to change from the current state.
	// otherwise we will add a delay
	checkvout := func(expected float64, msg string, pointKey string) {
		if lastvout == expected {
			// vout is not changing, so delay here to make sure the rule
			// has time to run before we check the result
			time.Sleep(time.Millisecond * 75)
		}

		start := time.Now()
		for {
			if voutGet().Value[pointKey] == expected {
				lastvout = expected
				// all is well
				break
			}
			if time.Since(start) > time.Second {
				t.Fatalf("vout failed, expected: %v, test: %v", expected, msg)
			}
			<-time.After(time.Millisecond * 10)
		}
	}

	sendPoint := func(id string, point data.Point) {
		point.Origin = "test"
		err = client.SendNodePoint(nc, id, point, true)

		if err != nil {
			t.Errorf("Error sending point: %v", err)
		}
	}

	/*
		leave everything enabled and toggle vin and watch vout toggle -- same as the TestRules() function.
		This ensures that your test is setup correctly.
	*/
	// set vin and look for vout to change
	sendPoint(vin.ID, data.Point{Type: data.PointTypeValue, Value: 1})

	checkvout(1, "set vin and look for vout to change", "0")

	// clear vin and look for vout to change
	sendPoint(vin.ID, data.Point{Type: data.PointTypeValue, Value: 0})

	checkvout(0, "clear vin and look for vout to change", "0")

	/*
		disable rule, set vin and verify vout does not get set. Then clear vin.
	*/
	// disable rule
	sendPoint(r.ID, data.Point{Type: data.PointTypeDisabled, Value: 1})

	sendPoint(vin.ID, data.Point{Type: data.PointTypeValue, Value: 1})

	// verify vout does not get set
	checkvout(0, "disable rule, set vin and verify vout does not get set", "0")

	//clear vin
	sendPoint(vin.ID, data.Point{Type: data.PointTypeValue, Value: 0})

	/*
		enable rule, and disable condition. set vin and verify vout does not get set. Clear vin.
	*/
	// enable rule
	sendPoint(r.ID, data.Point{Type: data.PointTypeDisabled, Value: 0})

	// disable condition
	sendPoint(c.ID, data.Point{Type: data.PointTypeDisabled, Value: 1})

	//set vin
	sendPoint(vin.ID, data.Point{Type: data.PointTypeValue, Value: 1})

	// if the rule client is broken.
	checkvout(0, "enable rule, and disable condition. set vin and verify vout does not get set", "0")

	//clear vin
	sendPoint(vin.ID, data.Point{Type: data.PointTypeValue, Value: 0})

	/*
		enable condition, and disable action. set vin and verify vout does not get set. Clear vin.
	*/

	// enable condition
	sendPoint(c.ID, data.Point{Type: data.PointTypeDisabled, Value: 0})

	//disable action
	sendPoint(a.ID, data.Point{Type: data.PointTypeDisabled, Value: 1})

	//set vin
	sendPoint(vin.ID, data.Point{Type: data.PointTypeValue, Value: 1})

	checkvout(0, "enable condition, and disable action. set vin and verify vout does not get set.", "0")

	//clear vin
	sendPoint(vin.ID, data.Point{Type: data.PointTypeValue, Value: 0})

	/*
		enable action, set vin, then disable rule. verify vout gets cleared.
	*/

	//enable action
	sendPoint(a.ID, data.Point{Type: data.PointTypeDisabled, Value: 0})

	//set vin
	sendPoint(vin.ID, data.Point{Type: data.PointTypeValue, Value: 1})

	// verify vout gets set
	checkvout(1, "enable action, set vin, then disable rule. verify vout gets set.", "0")

	//disable rule
	sendPoint(r.ID, data.Point{Type: data.PointTypeDisabled, Value: 1})

	checkvout(0, "enable action, set vin, then disable rule. verify vout gets cleared.", "0")

	/*
		enable rule, and verify vout gets set.
	*/

	//enable rule
	sendPoint(r.ID, data.Point{Type: data.PointTypeDisabled, Value: 0})

	// verify vout gets set
	checkvout(1, "enable rule, and verify vout gets set.", "0")

	/*
		disable condition, and verify vout gets cleared.
	*/

	//disable condition
	sendPoint(c.ID, data.Point{Type: data.PointTypeDisabled, Value: 1})

	// verify vout gets cleared.
	checkvout(0, "disable condition, and verify vout gets cleared.", "0")

	/*
		enable condition, and verify vout gets set.
	*/

	//enable condition
	sendPoint(c.ID, data.Point{Type: data.PointTypeDisabled, Value: 0})

	// verify vout gets set.
	checkvout(1, "enable condition, and verify vout gets set.", "0")

}

/*
if one condition is active and the 2nd condition is disabled, the rule fires
if both conditions are disabled, the rule is inactive.
*/
func TestMultipleConditions(t *testing.T) {
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

	// send test nodes to Db
	vin2 := client.Variable{
		ID:          "ID-varin2",
		Parent:      root.ID,
		Description: "var in2",
	}

	err = client.SendNodeType(nc, vin2, "test")
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

	// we don't want c2 to ever go active, so set the NodeID to some
	// bogus value
	c2 := client.Condition{
		ID:            "ID-condition2",
		Parent:        r.ID,
		Description:   "cond vin high",
		ConditionType: data.PointValuePointValue,
		PointType:     data.PointTypeValue,
		ValueType:     data.PointValueOnOff,
		NodeID:        vin2.ID,
		Operator:      data.PointValueEqual,
		Value:         1,
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

	if voutGet().Value["0"] != 0 {
		t.Fatal("initial vout value is not correct")
	}

	// wait for rule to get set up
	time.Sleep(250 * time.Millisecond)

	lastvout := float64(0)

	// set change to true if you are execting vout to change from the current state.
	// otherwise we will add a delay
	checkvout := func(expected float64, msg string) {
		if lastvout == expected {
			// vout is not changing, so delay here to make sure the rule
			// has time to run before we check the result
			time.Sleep(time.Millisecond * 75)
		}

		start := time.Now()
		for {
			if voutGet().Value["0"] == expected {
				lastvout = expected
				// all is well
				break
			}
			if time.Since(start) > time.Second {
				t.Fatalf("vout failed, expected: %v, test: %v", expected, msg)
			}
			<-time.After(time.Millisecond * 10)
		}
	}

	sendPoint := func(id string, point data.Point) {
		point.Origin = "test"
		err = client.SendNodePoint(nc, id, point, true)

		if err != nil {
			t.Errorf("Error sending point: %v", err)
		}
	}

	//	if one condition is active and the 2nd condition is inactive, the rule should not fire
	sendPoint(vin.ID, data.Point{Type: data.PointTypeValue, Value: 1})

	checkvout(0, "1st active, 2nd inactive")

	// if both conditions are active the rule should fire

	sendPoint(vin2.ID, data.Point{Type: data.PointTypeValue, Value: 1})

	checkvout(1, "both active")

	// if both conditions are active but disabled, the rule is inactive.

	sendPoint(c.ID, data.Point{Type: data.PointTypeDisabled, Value: 1})
	sendPoint(c2.ID, data.Point{Type: data.PointTypeDisabled, Value: 1})

	checkvout(0, "both active and disabled")
}

/*
Test PointKey of Action Node.
*/
func TestActionPointKey(t *testing.T) {
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
		PointKey:    "1",
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
		PointKey:    "1",
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

	if voutGet().Value["1"] != 0 {
		t.Fatal("initial vout value is not correct")
	}

	// wait for rule to get set up
	time.Sleep(250 * time.Millisecond)

	lastvout := float64(0)

	// set change to true if you are execting vout to change from the current state.
	// otherwise we will add a delay
	checkvout := func(expected float64, msg string, pointKey string) {
		if lastvout == expected {
			// vout is not changing, so delay here to make sure the rule
			// has time to run before we check the result
			time.Sleep(time.Millisecond * 75)
		}

		start := time.Now()
		for {
			if voutGet().Value[pointKey] == expected {
				lastvout = expected
				// all is well
				break
			}
			if time.Since(start) > time.Second {
				t.Fatalf("vout failed, expected: %v, test: %v", expected, msg)
			}
			<-time.After(time.Millisecond * 10)
		}
	}

	sendPoint := func(id string, point data.Point) {
		point.Origin = "test"
		err = client.SendNodePoint(nc, id, point, true)

		if err != nil {
			t.Errorf("Error sending point: %v", err)
		}
	}

	// check if point is set correctly.
	sendPoint(vin.ID, data.Point{Type: data.PointTypeValue, Value: 1})

	checkvout(1, "1st active, 2nd inactive", "1")

	// check if point is cleared correctly
	sendPoint(vin.ID, data.Point{Type: data.PointTypeValue, Value: 0})

	checkvout(0, "1st active, 2nd inactive", "1")
}
