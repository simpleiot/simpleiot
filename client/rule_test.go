package client_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

type ruleTestServer struct {
	t         *testing.T
	root      data.NodeEdge
	nc        *nats.Conn
	stop      func()
	vin       client.Variable
	vin2      client.Variable
	vout      client.Variable
	r         client.Rule
	c         client.Condition
	c2        client.Condition
	a         client.Action
	a2        client.ActionInactive
	voutGet   func() client.Variable
	voutStop  func()
	lastvout  float64
	lastCheck string
}

func (rts *ruleTestServer) checkVout(expected float64, msg string, pointKey string) {
	rts.lastCheck = msg
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

// checkVoutStays asserts the output holds its current value for the whole
// window. Most of the rule timing tests take this shape -- nothing should
// happen yet.
func (rts *ruleTestServer) checkVoutStays(expected float64, window time.Duration, msg string, pointKey string) {
	rts.lastCheck = msg

	start := time.Now()
	for time.Since(start) < window {
		if got := rts.voutGet().Value[pointKey]; got != expected {
			rts.t.Fatalf("vout changed to %v during %v, expected it to stay %v, test: %v",
				got, window, expected, msg)
		}
		<-time.After(time.Millisecond * 10)
	}

	rts.lastvout = expected
}

func (rts *ruleTestServer) sendPoint(id string, point data.Point) {
	point.Origin = "test"
	err := client.SendNodePoint(rts.nc, id, point, true)

	if err != nil {
		rts.t.Errorf("Error sending point: %v, last check: %v", err, rts.lastCheck)
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
	r.sendPoint(r.vin.ID, data.NewPointFloat(data.PointTypeValue, "", 1))
	r.checkVout(1, "look for vout to change after set vin", "0")

	// clear vin and look for vout to change
	r.sendPoint(r.vin.ID, data.NewPointFloat(data.PointTypeValue, "", 0))
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

	// everything enabled, set vin and look for vout to change
	r.sendPoint(r.vin.ID, data.NewPointFloat(data.PointTypeValue, "", 1))
	r.checkVout(1, "set vin and look for vout to change", "0")

	// everything enabled, clear vin and look for vout to change
	r.sendPoint(r.vin.ID, data.NewPointFloat(data.PointTypeValue, "", 0))
	r.checkVout(0, "clear vin and look for vout to change", "0")

	// disable rule, set vin and verify vout does not get set. Then clear vin.
	r.sendPoint(r.r.ID, data.NewPointFloat(data.PointTypeDisabled, "", 1))
	r.sendPoint(r.vin.ID, data.NewPointFloat(data.PointTypeValue, "", 1))
	r.checkVout(0, "disable rule, set vin and verify vout does not get set", "0")
	r.sendPoint(r.vin.ID, data.NewPointFloat(data.PointTypeValue, "", 0))

	// enable rule, and disable condition. set vin and verify vout does not get set.
	r.sendPoint(r.r.ID, data.NewPointFloat(data.PointTypeDisabled, "", 0))
	r.sendPoint(r.c.ID, data.NewPointFloat(data.PointTypeDisabled, "", 1))
	r.sendPoint(r.vin.ID, data.NewPointFloat(data.PointTypeValue, "", 1))
	r.checkVout(0, "disable condition and verify vout does not get set", "0")

	// enable condition and verify vout gets set again. Clear vin.
	r.sendPoint(r.c.ID, data.NewPointFloat(data.PointTypeDisabled, "", 0))
	r.checkVout(1, "re-enable condition and verify vout is set.", "0")
	r.sendPoint(r.vin.ID, data.NewPointFloat(data.PointTypeValue, "", 0))

	// enable condition, and disable action. set vin and verify vout does not get set. Clear vin.
	r.sendPoint(r.a.ID, data.NewPointFloat(data.PointTypeDisabled, "", 1))
	r.sendPoint(r.vin.ID, data.NewPointFloat(data.PointTypeValue, "", 1))
	r.checkVout(0, "disable action and verify vout does not get set.", "0")
	r.sendPoint(r.vin.ID, data.NewPointFloat(data.PointTypeValue, "", 0))

	// 	enable action, set vin, then disable rule. verify vout gets cleared.
	r.sendPoint(r.a.ID, data.NewPointFloat(data.PointTypeDisabled, "", 0))
	r.sendPoint(r.vin.ID, data.NewPointFloat(data.PointTypeValue, "", 1))
	r.checkVout(1, "disable rule, initial state", "0")
	r.sendPoint(r.r.ID, data.NewPointFloat(data.PointTypeDisabled, "", 1))
	r.checkVout(0, "disable rule, vout cleared.", "0")

	// enable rule, and verify vout gets set.
	r.sendPoint(r.r.ID, data.NewPointFloat(data.PointTypeDisabled, "", 0))
	r.checkVout(1, "enable rule, and verify vout gets set.", "0")

	// disable condition, and verify vout gets cleared.
	r.sendPoint(r.c.ID, data.NewPointFloat(data.PointTypeDisabled, "", 1))
	r.checkVout(0, "disable condition, and verify vout gets cleared.", "0")

	// enable condition, and verify vout gets set.
	r.sendPoint(r.c.ID, data.NewPointFloat(data.PointTypeDisabled, "", 0))
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

	// both conditions enabled
	// if one condition is active and the 2nd condition is inactive, the rule should not fire
	r.sendPoint(r.vin.ID, data.NewPointFloat(data.PointTypeValue, "", 1))
	r.checkVout(0, "1st active, 2nd inactive", "0")

	// if both conditions are active the rule should fire
	r.sendPoint(r.vin2.ID, data.NewPointFloat(data.PointTypeValue, "", 1))
	r.checkVout(1, "both active", "0")

	// if one condition is disabled, the rule should still fire because
	// the enabled condition is still active
	r.sendPoint(r.c.ID, data.NewPointFloat(data.PointTypeDisabled, "", 1))
	r.checkVout(1, "one condition enabled and action", "0")

	// if both conditions are active but disabled, the rule is inactive.
	r.sendPoint(r.c.ID, data.NewPointFloat(data.PointTypeDisabled, "", 1))
	r.sendPoint(r.c2.ID, data.NewPointFloat(data.PointTypeDisabled, "", 1))
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
	r.sendPoint(r.a.ID, data.NewPointString(data.PointTypePointKey, "", "1"))
	r.sendPoint(r.a2.ID, data.NewPointString(data.PointTypePointKey, "", "1"))

	defer r.stop()
	defer r.voutStop()

	r.checkVout(0, "initial value", "1")

	// check if point is set correctly.
	r.sendPoint(r.vin.ID, data.NewPointFloat(data.PointTypeValue, "", 1))
	r.checkVout(1, "should be high", "1")

	// check if point is cleared correctly
	r.sendPoint(r.vin.ID, data.NewPointFloat(data.PointTypeValue, "", 0))

	r.checkVout(0, "should be low", "1")
}

// countPoints counts points of a given type sent to a node. Points are sent to
// the p.<id> subject, so counting means subscribing rather than reading the
// current value -- a notification point uses a fixed key, so a second
// notification overwrites the first.
func (rts *ruleTestServer) countPoints(id, typ string) (count func() int, stop func()) {
	var lock sync.Mutex
	c := 0

	unsub, err := client.SubscribePoints(rts.nc, id, func(points []data.Point) {
		lock.Lock()
		defer lock.Unlock()
		for _, p := range points {
			if p.Type == typ {
				c++
			}
		}
	})

	if err != nil {
		rts.t.Fatalf("Error subscribing to points for %v: %v", id, err)
	}

	return func() int {
		lock.Lock()
		defer lock.Unlock()
		return c
	}, unsub
}

// addNotifyAction adds a notify action to the rule so notifications can be
// counted alongside the setValue action the harness already installs.
func (rts *ruleTestServer) addNotifyAction(id string) client.Action {
	a := client.Action{
		ID:          id,
		Parent:      rts.r.ID,
		Description: "notify " + id,
		Action:      data.PointValueNotify,
	}

	err := client.SendNodeType(rts.nc, a, "test")
	if err != nil {
		rts.t.Fatalf("Error sending notify action: %v", err)
	}

	// give the rule client time to pick up the new child
	time.Sleep(250 * time.Millisecond)

	return a
}

/*
Actions run when the rule changes state, and only then. Editing a rule, a
condition, or an action while the rule is active must not re-run the actions or
re-send the notification.
*/
func TestRuleTransitionsOnly(t *testing.T) {
	r, err := setupRuleTest(t, 1)
	if err != nil {
		t.Fatal("Rule test setup failed: ", err)
	}

	defer r.stop()
	defer r.voutStop()

	r.addNotifyAction("ID-action-notify")

	notifyCount, notifyStop := r.countPoints(r.r.ID, data.PointTypeNotification)
	defer notifyStop()

	voutCount, voutStop := r.countPoints(r.vout.ID, data.PointTypeValue)
	defer voutStop()

	r.checkVout(0, "initial value", "0")

	// set vin and look for vout to change
	r.sendPoint(r.vin.ID, data.NewPointFloat(data.PointTypeValue, "", 1))
	r.checkVout(1, "look for vout to change after set vin", "0")

	time.Sleep(100 * time.Millisecond)

	if notifyCount() != 1 {
		t.Fatalf("expected 1 notification after the rule went active, got %v", notifyCount())
	}

	notifyBefore := notifyCount()
	voutBefore := voutCount()

	// edit the rule, the condition, and the action while the rule is active
	r.sendPoint(r.r.ID, data.NewPointString(data.PointTypeDescription, "", "test rule edited"))
	r.sendPoint(r.c.ID, data.NewPointString(data.PointTypeDescription, "", "cond edited"))
	r.sendPoint(r.a.ID, data.NewPointString(data.PointTypeDescription, "", "action edited"))

	time.Sleep(300 * time.Millisecond)

	if notifyCount() != notifyBefore {
		t.Errorf("editing an active rule sent %v extra notifications",
			notifyCount()-notifyBefore)
	}

	if voutCount() != voutBefore {
		t.Errorf("editing an active rule wrote the output %v extra times",
			voutCount()-voutBefore)
	}
}

/*
Disabling an active rule runs the inactive actions exactly once, and further
edits while the rule is disabled run nothing.
*/
func TestRuleDisableTransition(t *testing.T) {
	r, err := setupRuleTest(t, 1)
	if err != nil {
		t.Fatal("Rule test setup failed: ", err)
	}

	defer r.stop()
	defer r.voutStop()

	voutCount, voutStop := r.countPoints(r.vout.ID, data.PointTypeValue)
	defer voutStop()

	r.checkVout(0, "initial value", "0")

	r.sendPoint(r.vin.ID, data.NewPointFloat(data.PointTypeValue, "", 1))
	r.checkVout(1, "rule active", "0")

	countBefore := voutCount()

	r.sendPoint(r.r.ID, data.NewPointFloat(data.PointTypeDisabled, "", 1))
	r.checkVout(0, "disabling an active rule runs the inactive actions", "0")

	// further edits while disabled run nothing
	r.sendPoint(r.r.ID, data.NewPointString(data.PointTypeDescription, "", "edited while disabled"))

	time.Sleep(300 * time.Millisecond)

	if voutCount() != countBefore+1 {
		t.Errorf("expected 1 output write on disable, got %v", voutCount()-countBefore)
	}
}

// minutes converts the fractional minutes the rule timing points are expressed
// in. The timing tests use hundredths of a minute -- 0.01 minutes is 600ms --
// which keeps the suite fast while still exercising real durations.
func minutes(m float64) time.Duration {
	return time.Duration(m * float64(time.Minute))
}

/*
minActive is a pending period: the condition has to hold continuously for that
long before the rule goes active.
*/
func TestRuleMinActive(t *testing.T) {
	r, err := setupRuleTest(t, 1)
	if err != nil {
		t.Fatal("Rule test setup failed: ", err)
	}

	defer r.stop()
	defer r.voutStop()

	const minActive = 0.01 // 600ms

	r.sendPoint(r.c.ID, data.NewPointFloat(data.PointTypeMinActive, "", minActive))
	time.Sleep(150 * time.Millisecond)

	r.checkVout(0, "initial value", "0")

	// a spike that returns before the pending period expires never activates
	r.sendPoint(r.vin.ID, data.NewPointFloat(data.PointTypeValue, "", 1))
	r.checkVoutStays(0, minutes(minActive)/2, "spike does not activate immediately", "0")
	r.sendPoint(r.vin.ID, data.NewPointFloat(data.PointTypeValue, "", 0))
	r.checkVoutStays(0, minutes(minActive), "spike never activates the rule", "0")

	// a value that stays activates after the period, with no further points
	r.sendPoint(r.vin.ID, data.NewPointFloat(data.PointTypeValue, "", 1))
	r.checkVoutStays(0, minutes(minActive)/2, "pending period is still running", "0")
	r.checkVout(1, "rule activates when the pending period expires", "0")

	// clearing has no pending period of its own
	r.sendPoint(r.vin.ID, data.NewPointFloat(data.PointTypeValue, "", 0))
	r.checkVout(0, "rule clears immediately", "0")
}

/*
Changing minActive while a pending period is running applies the new value
against the time the input changed.
*/
func TestRuleMinActiveChanged(t *testing.T) {
	r, err := setupRuleTest(t, 1)
	if err != nil {
		t.Fatal("Rule test setup failed: ", err)
	}

	defer r.stop()
	defer r.voutStop()

	r.sendPoint(r.c.ID, data.NewPointFloat(data.PointTypeMinActive, "", 0.2)) // 12s
	time.Sleep(150 * time.Millisecond)

	r.checkVout(0, "initial value", "0")

	r.sendPoint(r.vin.ID, data.NewPointFloat(data.PointTypeValue, "", 1))
	r.checkVoutStays(0, 200*time.Millisecond, "long pending period is running", "0")

	// shorten the pending period to one that has already elapsed
	r.sendPoint(r.c.ID, data.NewPointFloat(data.PointTypeMinActive, "", 0.001))
	r.checkVout(1, "shortening minActive while pending activates the rule", "0")
}

/*
Disabling a rule clears any pending period, so enabling it again starts the
wait over.
*/
func TestRuleMinActiveDisabled(t *testing.T) {
	r, err := setupRuleTest(t, 1)
	if err != nil {
		t.Fatal("Rule test setup failed: ", err)
	}

	defer r.stop()
	defer r.voutStop()

	const minActive = 0.02 // 1.2s

	r.sendPoint(r.c.ID, data.NewPointFloat(data.PointTypeMinActive, "", minActive))
	r.sendPoint(r.r.ID, data.NewPointFloat(data.PointTypeDisabled, "", 1))
	time.Sleep(150 * time.Millisecond)

	// the input goes true while the rule is disabled, so nothing is counting
	r.sendPoint(r.vin.ID, data.NewPointFloat(data.PointTypeValue, "", 1))
	r.checkVoutStays(0, minutes(minActive), "a disabled rule counts nothing", "0")

	// enabling the rule starts the pending period over from an input that is
	// already true, which needs a fresh point to be seen at all
	r.sendPoint(r.r.ID, data.NewPointFloat(data.PointTypeDisabled, "", 0))
	r.sendPoint(r.vin.ID, data.NewPointFloat(data.PointTypeValue, "", 1))
	r.checkVoutStays(0, minutes(minActive)/2, "pending period restarted", "0")
	r.checkVout(1, "rule activates after the restarted pending period", "0")
}

/*
minInactive holds a condition active until its input has been clear for the
full duration, so an input oscillating across a threshold is one incident and
one notification rather than one per cycle.
*/
func TestRuleMinInactive(t *testing.T) {
	r, err := setupRuleTest(t, 1)
	if err != nil {
		t.Fatal("Rule test setup failed: ", err)
	}

	defer r.stop()
	defer r.voutStop()

	r.addNotifyAction("ID-action-notify")

	notifyCount, notifyStop := r.countPoints(r.r.ID, data.PointTypeNotification)
	defer notifyStop()

	const minInactive = 0.02 // 1.2s

	r.sendPoint(r.c.ID, data.NewPointFloat(data.PointTypeMinInactive, "", minInactive))
	time.Sleep(150 * time.Millisecond)

	r.checkVout(0, "initial value", "0")

	// the rule activates with no pending period
	r.sendPoint(r.vin.ID, data.NewPointFloat(data.PointTypeValue, "", 1))
	r.checkVout(1, "rule activates immediately", "0")

	// oscillate faster than the hold; the rule must stay active throughout
	for i := 0; i < 3; i++ {
		r.sendPoint(r.vin.ID, data.NewPointFloat(data.PointTypeValue, "", 0))
		r.checkVoutStays(1, minutes(minInactive)/3, "hold keeps the rule active", "0")
		r.sendPoint(r.vin.ID, data.NewPointFloat(data.PointTypeValue, "", 1))
		r.checkVoutStays(1, minutes(minInactive)/3, "returning cancels the hold", "0")
	}

	if notifyCount() != 1 {
		t.Errorf("expected 1 notification for one incident, got %v", notifyCount())
	}

	// once the input stays clear, the condition deactivates on its own
	r.sendPoint(r.vin.ID, data.NewPointFloat(data.PointTypeValue, "", 0))
	r.checkVoutStays(1, minutes(minInactive)/2, "hold is still running", "0")
	r.checkVout(0, "rule clears once the input has been clear for minInactive", "0")
}
