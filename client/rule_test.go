package client_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

type ruleTestServer struct {
	t        *testing.T
	root     data.NodeEdge
	nc       *nats.Conn
	stop     func()
	vin      client.Variable
	vin2     client.Variable
	vout     client.Variable
	r        client.Rule
	c        client.Condition
	c2       client.Condition
	a        client.Action
	a2       client.ActionInactive
	voutGet  func() client.Variable
	voutStop func()
	lastvout float64
}

func (rts *ruleTestServer) checkVout(expected float64, msg string, pointKey string) {

	if rts.lastvout == expected {
		// vout is not changing, so delay here to make sure the rule
		// has time to run before we check the result
		time.Sleep(time.Millisecond * 75)
	}

	start := time.Now()
	for {
		if rts.voutGet().Value[pointKey] == expected {
			rts.lastvout = expected
			// all is well
			break
		}
		if time.Since(start) > time.Second {
			rts.t.Fatalf("vout failed, expected: %v, test: %v", expected, msg)
		}
		<-time.After(time.Millisecond * 10)
	}
}

func (rts *ruleTestServer) sendPoint(id string, point data.Point) {
	point.Origin = "test"
	err := client.SendNodePoint(rts.nc, id, point, true)

	if err != nil {
		rts.t.Errorf("Error sending point: %v", err)
	}
}

func setupRuleTest(t *testing.T, numConditions int) (ruleTestServer, error) {

	var r ruleTestServer
	var err error

	r.t = t

	r.nc, r.root, r.stop, err = server.TestServer()

	if err != nil {
		return r, fmt.Errorf("Error starting test server: %w", err)
	}
	// send test nodes to Db
	r.vin = client.Variable{
		ID:          "ID-varin",
		Parent:      r.root.ID,
		Description: "var in",
	}

	err = client.SendNodeType(r.nc, r.vin, "test")
	if err != nil {
		return r, fmt.Errorf("Error sending vin node: %w", err)
	}

	r.vout = client.Variable{
		ID:          "ID-varout",
		Parent:      r.root.ID,
		Description: "var out",
	}

	err = client.SendNodeType(r.nc, r.vout, "test")
	if err != nil {
		return r, fmt.Errorf("Error sending vout node: %w", err)
	}

	r.r = client.Rule{
		ID:          "ID-rule",
		Parent:      r.root.ID,
		Description: "test rule",
		Disabled:    false,
	}

	err = client.SendNodeType(r.nc, r.r, "test")
	if err != nil {
		return r, fmt.Errorf("Error sending r node: %w", err)
	}

	r.c = client.Condition{
		ID:            "ID-condition",
		Parent:        r.r.ID,
		Description:   "cond vin high",
		ConditionType: data.PointValuePointValue,
		PointType:     data.PointTypeValue,
		ValueType:     data.PointValueOnOff,
		NodeID:        r.vin.ID,
		Operator:      data.PointValueEqual,
		Value:         1,
	}

	err = client.SendNodeType(r.nc, r.c, "test")
	if err != nil {
		return r, fmt.Errorf("Error sending c node: %w", err)
	}

	if numConditions > 1 {
		// send test nodes to Db
		r.vin2 = client.Variable{
			ID:          "ID-varin2",
			Parent:      r.root.ID,
			Description: "var in2",
		}

		err = client.SendNodeType(r.nc, r.vin2, "test")
		if err != nil {
			return r, fmt.Errorf("Error sending vin2 node: %w", err)
		}

		r.c2 = client.Condition{
			ID:            "ID-condition2",
			Parent:        r.r.ID,
			Description:   "cond vin2 high",
			ConditionType: data.PointValuePointValue,
			PointType:     data.PointTypeValue,
			ValueType:     data.PointValueOnOff,
			NodeID:        r.vin2.ID,
			Operator:      data.PointValueEqual,
			Value:         1,
		}

		err = client.SendNodeType(r.nc, r.c2, "test")
		if err != nil {
			return r, fmt.Errorf("Error sending c node: %w", err)
		}
	}

	r.a = client.Action{
		ID:          "ID-action-active",
		Parent:      r.r.ID,
		Description: "action active",
		Action:      data.PointValueSetValue,
		PointType:   data.PointTypeValue,
		NodeID:      r.vout.ID,
		Value:       1,
	}

	err = client.SendNodeType(r.nc, r.a, "test")
	if err != nil {
		return r, fmt.Errorf("Error sending a node: %w", err)
	}

	// FIXME:
	// this delay is required to work around a bug in the manager
	// where it is resetting and does not see the ActionInactive points
	// See https://github.com/simpleiot/simpleiot/issues/630
	// the tools/test-rules.sh script can be used to test a fix for this
	// problem
	time.Sleep(100 * time.Millisecond)

	r.a2 = client.ActionInactive{
		ID:          "ID-action-inactive",
		Parent:      r.r.ID,
		Description: "action inactive",
		Action:      data.PointValueSetValue,
		PointType:   data.PointTypeValue,
		NodeID:      r.vout.ID,
		Value:       0,
	}

	err = client.SendNodeType(r.nc, r.a2, "test")
	if err != nil {
		return r, fmt.Errorf("Error sending a2 node: %w", err)
	}

	// set up a node watcher to watch the output variable
	r.voutGet, r.voutStop, err = client.NodeWatcher[client.Variable](r.nc, r.vout.ID, r.vout.Parent)

	if err != nil {
		return r, fmt.Errorf("Error setting up watcher: %w", err)
	}

	// wait for rule to get set up
	time.Sleep(250 * time.Millisecond)

	return r, nil
}

// TestRules populates a rule in the system that watches
// a variable and when set, sets another variable. This
// tests out the basic rule logic.
func TestRule(t *testing.T) {
	r, err := setupRuleTest(t, 1)
	if err != nil {
		t.Fatal("Rule test setup failed: ", err)
	}

	defer r.stop()
	defer r.voutStop()

	r.checkVout(0, "initial value", "0")

	// set vin and look for vout to change
	r.sendPoint(r.vin.ID, data.Point{Type: data.PointTypeValue, Value: 1})
	r.checkVout(1, "look for vout to change after set vin", "0")

	// clear vin and look for vout to change
	r.sendPoint(r.vin.ID, data.Point{Type: data.PointTypeValue, Value: 0})
	r.checkVout(0, "look for vout to clear", "0")
}

/*
leave everything enabled and toggle vin and watch vout toggle -- same as the TestRules() function. This ensures that your test is setup correctly.
disable rule, set vin and verify vout does not get set. Then clear vin.
- enable rule, and disable condition. set vin and verify vout does not get set. Clear vin.
- enable condition, and disable action. set vin and verify vout does not get set. Clear vin.
- enable action, set vin, then disable rule. verify vout gets cleared.
- enable rule, and verify vout gets set.
- disable condition, and verify vout gets cleared.
- enable condition, and verify vout gets set.
*/
func TestRuleDisabled(t *testing.T) {
	r, err := setupRuleTest(t, 1)
	if err != nil {
		t.Fatal("Rule test setup failed: ", err)
	}

	defer r.stop()
	defer r.voutStop()

	r.checkVout(0, "check initial state", "0")

	// leave everything enabled and toggle vin and watch vout toggle -- same as the TestRules() function.
	// This ensures that your test is setup correctly.

	// set vin and look for vout to change
	r.sendPoint(r.vin.ID, data.Point{Type: data.PointTypeValue, Value: 1})
	r.checkVout(1, "set vin and look for vout to change", "0")

	// clear vin and look for vout to change
	r.sendPoint(r.vin.ID, data.Point{Type: data.PointTypeValue, Value: 0})
	r.checkVout(0, "clear vin and look for vout to change", "0")

	// disable rule, set vin and verify vout does not get set. Then clear vin.

	// disable rule
	r.sendPoint(r.r.ID, data.Point{Type: data.PointTypeDisabled, Value: 1})
	r.sendPoint(r.vin.ID, data.Point{Type: data.PointTypeValue, Value: 1})

	// verify vout does not get set
	r.checkVout(0, "disable rule, set vin and verify vout does not get set", "0")

	//clear vin
	r.sendPoint(r.vin.ID, data.Point{Type: data.PointTypeValue, Value: 0})

	// enable rule, and disable condition. set vin and verify vout does not get set. Clear vin.

	// enable rule
	r.sendPoint(r.r.ID, data.Point{Type: data.PointTypeDisabled, Value: 0})

	// disable condition
	r.sendPoint(r.c.ID, data.Point{Type: data.PointTypeDisabled, Value: 1})

	//set vin
	r.sendPoint(r.vin.ID, data.Point{Type: data.PointTypeValue, Value: 1})

	// if the rule client is broken.
	r.checkVout(0, "enable rule, and disable condition. set vin and verify vout does not get set", "0")

	//clear vin
	r.sendPoint(r.vin.ID, data.Point{Type: data.PointTypeValue, Value: 0})

	/*
		enable condition, and disable action. set vin and verify vout does not get set. Clear vin.
	*/

	// enable condition
	r.sendPoint(r.c.ID, data.Point{Type: data.PointTypeDisabled, Value: 0})

	//disable action
	r.sendPoint(r.a.ID, data.Point{Type: data.PointTypeDisabled, Value: 1})

	//set vin
	r.sendPoint(r.vin.ID, data.Point{Type: data.PointTypeValue, Value: 1})
	r.checkVout(0, "enable condition, and disable action. set vin and verify vout does not get set.", "0")

	//clear vin
	r.sendPoint(r.vin.ID, data.Point{Type: data.PointTypeValue, Value: 0})

	// 	enable action, set vin, then disable rule. verify vout gets cleared.

	//enable action
	r.sendPoint(r.a.ID, data.Point{Type: data.PointTypeDisabled, Value: 0})

	//set vin
	r.sendPoint(r.vin.ID, data.Point{Type: data.PointTypeValue, Value: 1})

	// verify vout gets set
	r.checkVout(1, "enable action, set vin, then disable rule. verify vout gets set.", "0")

	//disable rule
	r.sendPoint(r.r.ID, data.Point{Type: data.PointTypeDisabled, Value: 1})
	r.checkVout(0, "enable action, set vin, then disable rule. verify vout gets cleared.", "0")

	// enable rule, and verify vout gets set.

	//enable rule
	r.sendPoint(r.r.ID, data.Point{Type: data.PointTypeDisabled, Value: 0})

	// verify vout gets set
	r.checkVout(1, "enable rule, and verify vout gets set.", "0")

	// disable condition, and verify vout gets cleared.

	//disable condition
	r.sendPoint(r.c.ID, data.Point{Type: data.PointTypeDisabled, Value: 1})

	// verify vout gets cleared.
	r.checkVout(0, "disable condition, and verify vout gets cleared.", "0")

	// enable condition, and verify vout gets set.

	//enable condition
	r.sendPoint(r.c.ID, data.Point{Type: data.PointTypeDisabled, Value: 0})

	// verify vout gets set.
	r.checkVout(1, "enable condition, and verify vout gets set.", "0")
}

/*
if one condition is active and the 2nd condition is disabled, the rule fires
if both conditions are disabled, the rule is inactive.
*/
func TestRuleMultipleConditions(t *testing.T) {

	r, err := setupRuleTest(t, 2)
	if err != nil {
		t.Fatal("Rule test setup failed: ", err)
	}

	defer r.stop()
	defer r.voutStop()

	r.checkVout(0, "initial condition", "0")

	//	if one condition is active and the 2nd condition is inactive, the rule should not fire
	r.sendPoint(r.vin.ID, data.Point{Type: data.PointTypeValue, Value: 1})
	r.checkVout(0, "1st active, 2nd inactive", "0")

	// if both conditions are active the rule should fire
	r.sendPoint(r.vin2.ID, data.Point{Type: data.PointTypeValue, Value: 1})
	r.checkVout(1, "both active", "0")

	// if both conditions are active but disabled, the rule is inactive.

	r.sendPoint(r.c.ID, data.Point{Type: data.PointTypeDisabled, Value: 1})
	r.sendPoint(r.c2.ID, data.Point{Type: data.PointTypeDisabled, Value: 1})
	r.checkVout(0, "both active and disabled", "0")
}

/*
Test PointKey of Action Node.
*/
func TestRuleActionPointKey(t *testing.T) {
	r, err := setupRuleTest(t, 1)
	if err != nil {
		t.Fatal("Rule test setup failed: ", err)
	}

	// we are setting the an action with key set to "1", so modify the rule
	r.sendPoint(r.a.ID, data.Point{Type: data.PointTypePointKey, Text: "1"})
	r.sendPoint(r.a2.ID, data.Point{Type: data.PointTypePointKey, Text: "1"})

	defer r.stop()
	defer r.voutStop()

	r.checkVout(0, "inital value", "1")

	// check if point is set correctly.
	r.sendPoint(r.vin.ID, data.Point{Type: data.PointTypeValue, Value: 1})
	r.checkVout(1, "should be high", "1")

	// check if point is cleared correctly
	r.sendPoint(r.vin.ID, data.Point{Type: data.PointTypeValue, Value: 0})

	r.checkVout(0, "should be low", "1")
}
