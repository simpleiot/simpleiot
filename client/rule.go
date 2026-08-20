package client

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/go-audio/wav"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// Rule represent a rule node config
type Rule struct {
	ID              string      `node:"id"`
	Parent          string      `node:"parent"`
	Description     string      `point:"description"`
	Disabled        bool        `point:"disabled"`
	Active          bool        `point:"active"`
	Error           string      `point:"error"`
	Conditions      []Condition `child:"condition"`
	Actions         []Action    `child:"action"`
	ActionsInactive []Action    `child:"actionInactive"`
}

func (r Rule) String() string {
	ret := fmt.Sprintf("Rule: %v\n", r.Description)
	ret += fmt.Sprintf("  active: %v\n", r.Active)
	ret += fmt.Sprintf("  Disabled: %v\n", r.Disabled)
	for _, c := range r.Conditions {
		ret += fmt.Sprintf("%v", c)
	}
	for _, a := range r.Actions {
		ret += fmt.Sprintf("  ACTION: %v", a)
	}

	for _, a := range r.ActionsInactive {
		ret += fmt.Sprintf("  ACTION Inactive: %v", a)
	}

	return ret
}

// Condition defines parameters to look for in a point or a schedule.
type Condition struct {
	// general parameters
	ID            string  `node:"id"`
	Parent        string  `node:"parent"`
	Description   string  `point:"description"`
	Disabled      bool    `point:"disabled"`
	ConditionType string  `point:"conditionType"`
	MinActive     float64 `point:"minActive"`
	Active        bool    `point:"active"`
	Error         string  `point:"error"`

	// used with point value rules
	NodeID     string  `point:"nodeID"`
	PointType  string  `point:"pointType"`
	PointKey   string  `point:"pointKey"`
	PointIndex int     `point:"pointIndex"`
	ValueType  string  `point:"valueType"`
	Operator   string  `point:"operator"`
	Value      float64 `point:"value"`
	ValueText  string  `point:"valueText"`

	// used with schedule rules
	Start    string   `point:"start"`
	End      string   `point:"end"`
	Weekdays []bool   `point:"weekday"`
	Dates    []string `point:"date"`
}

func (c Condition) String() string {
	value := ""
	switch c.ValueType {
	case data.PointValueOnOff:
		if c.Value == 0 {
			value = "off"
		} else {
			value = "on"
		}
	case data.PointValueNumber:
		value = strconv.FormatFloat(c.Value, 'f', 2, 64)
	case data.PointValueText:
		value = c.ValueText
	}

	var ret string

	switch c.ConditionType {
	case data.PointValuePointValue:
		ret = fmt.Sprintf("  COND: %v  Disabled: %v CTYPE:%v  VTYPE:%v  V:%v",
			c.Description, c.ConditionType, c.Disabled, c.ValueType, value)
		if c.NodeID != "" {
			ret += fmt.Sprintf("  NODEID:%v", c.NodeID)
		}
		if c.MinActive > 0 {
			ret += fmt.Sprintf("  MINACT:%v", c.MinActive)
		}
		ret += fmt.Sprintf("  A:%v", c.Active)
		ret += "\n"
	case data.PointValueSchedule:
		ret = fmt.Sprintf("  COND: %v  CTYPE:%v",
			c.Description, c.ConditionType)
		ret += fmt.Sprintf("  W:%v", c.Weekdays)
		ret += fmt.Sprintf("  D:%v", c.Dates)
		ret += "\n"

	default:
		ret = "Missing String case for condition"
	}
	return ret
}

// Action defines actions that can be taken if a rule is active.
type Action struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	Disabled    bool   `point:"disabled"`
	Active      bool   `point:"active"`
	Error       string `point:"error"`
	// Action: notify, setValue, playAudio
	Action    string `point:"action"`
	NodeID    string `point:"nodeID"`
	PointType string `point:"pointType"`
	PointKey  string `point:"pointKey"`
	// PointType: number, text, onOff
	ValueType string  `point:"valueType"`
	Value     float64 `point:"value"`
	ValueText string  `point:"valueText"`
	// the following are used for audio playback
	Channel  int    `point:"channel"`
	Device   string `point:"device"`
	FilePath string `point:"filePath"`
}

func (a Action) String() string {
	value := ""
	switch a.ValueType {
	case data.PointValueOnOff:
		if a.Value == 0 {
			value = "off"
		} else {
			value = "on"
		}
	case data.PointValueNumber:
		value = strconv.FormatFloat(a.Value, 'f', 2, 64)
	case data.PointValueText:
		value = a.ValueText
	}
	ret := fmt.Sprintf("%v  Disabled:%v ACT:%v  VTYPE:%v  V:%v",
		a.Description, a.Disabled, a.Action, a.ValueType, value)
	if a.NodeID != "" {
		ret += fmt.Sprintf("  NODEID:%v", a.NodeID)
	}
	if a.PointKey != "" && a.PointKey != "0" {
		ret += fmt.Sprintf(" K:%v", a.PointKey)
	}
	ret += fmt.Sprintf("  A:%v", a.Active)
	ret += "\n"
	return ret
}

// ActionInactive defines actions that can be taken if a rule is inactive.
// this is defined for use with the client.SendNodeType API
type ActionInactive struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	Active      bool   `point:"active"`
	// Action: notify, setValue, playAudio
	Action    string `point:"action"`
	NodeID    string `point:"nodeID"`
	PointType string `point:"pointType"`
	PointKey  string `point:"pointKey"`
	// PointType: number, text, onOff
	ValueType string  `point:"valueType"`
	Value     float64 `point:"value"`
	ValueText string  `point:"valueText"`
	// the following are used for audio playback
	Channel  int    `point:"channel"`
	Device   string `point:"device"`
	FilePath string `point:"filePath"`
}

// RuleClient is a SIOT client used to run rules
type RuleClient struct {
	nc            *nats.Conn
	config        Rule
	stop          chan struct{}
	newPoints     chan NewPoints
	newEdgePoints chan NewPoints
	newRulePoints chan NewPoints
	upSub         *nats.Subscription

	// actionState is the rule state the actions were last run for. Actions
	// run only when the rule changes state, so a configuration edit on an
	// active rule no longer re-runs them. It is seeded from the rule's
	// persisted active point in Run(), which means a client that starts up
	// and computes the state it was already in does not re-assert its
	// setValue outputs or re-send its notification.
	actionState bool

	// condState holds the in-process state of each condition, keyed by
	// condition node ID
	condState map[string]*condRuntime

	// lastTrigger is the node that last moved a condition, used to name the
	// node that fired the rule when an action runs from a timer rather than
	// from an inbound point
	lastTrigger string
}

// NewRuleClient constructor ...
func NewRuleClient(nc *nats.Conn, config Rule) Client {
	return &RuleClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan NewPoints),
		newEdgePoints: make(chan NewPoints),
		newRulePoints: make(chan NewPoints),
	}
}

// Run runs the main logic for this client and blocks until stopped
func (rc *RuleClient) Run() error {
	// the rule's active point is persisted, so the state the actions were
	// last run for is the state the rule resumes in
	rc.actionState = rc.config.Active

	// watch all points that flow through parent node
	// TODO: we should optimize this so we only watch the nodes
	// that are in the conditions
	subject := fmt.Sprintf("up.%v.>", rc.config.Parent)

	var err error
	rc.upSub, err = rc.nc.Subscribe(subject, func(msg *nats.Msg) {
		points, err := data.DecodePoints(msg.Data)
		if err != nil {
			log.Println("Error decoding points in rule upSub:", err)
			return
		}

		// find node ID for points
		// up.<parentId>.<nodeId>.<type>.<key> = 5 chunks
		chunks := strings.Split(msg.Subject, ".")
		if len(chunks) < 3 {
			log.Println("rule client up sub, malformed subject:", msg.Subject)
			return
		}

		rc.newRulePoints <- NewPoints{chunks[2], "", points}
	})

	if err != nil {
		return fmt.Errorf("Rule error subscribing to upsub: %v", err)
	}

	// TODO schedule ticker is a brute force way to do this
	// we could optimize at some point by creating a timer to expire
	// on the next schedule change
	scheduleTickTime := time.Second * 10
	scheduleTicker := time.NewTicker(scheduleTickTime)
	if !rc.hasSchedule() {
		scheduleTicker.Stop()
	}

	// deadlineTimer fires when a condition can change state on its own, with
	// no inbound point. It stays stopped whenever nothing is pending, so an
	// idle rule costs nothing.
	deadlineTimer := time.NewTimer(time.Hour)
	if !deadlineTimer.Stop() {
		<-deadlineTimer.C
	}

	armDeadline := func() {
		if !deadlineTimer.Stop() {
			select {
			case <-deadlineTimer.C:
			default:
			}
		}

		next := rc.nextDeadline()
		if next.IsZero() {
			return
		}

		wait := time.Until(next)
		if wait < 0 {
			wait = 0
		}

		deadlineTimer.Reset(wait)
	}

	// run evaluates the rule and acts on a change of state. points may be
	// empty, which is how a timer re-evaluates conditions with nothing new to
	// compare against.
	run := func(id string, pts data.Points) {
		var active bool

		defer armDeadline()

		if rc.config.Disabled {
			// a disabled rule is inactive, and publishing that keeps the state
			// the actions ran for and the state the UI shows in agreement
			rc.setRuleActive(false)

			// a disabled rule counts nothing, so any pending period is
			// cleared and starts over when the rule is enabled again
			rc.condState = nil
		} else {
			if len(pts) > 0 {
				rc.ruleUpdateConditions(id, pts)
			}

			rc.ruleApplyHeldState(time.Now())
			active = rc.ruleComputeActive()
		}

		if id == "" {
			// the rule ran from a timer rather than from an inbound point;
			// name the node that last moved a condition as the trigger
			id = rc.lastTrigger
		}

		if id == "" {
			id = rc.config.ID
		}

		if active == rc.actionState {
			// actions run on state transitions only
			return
		}

		rc.actionState = active

		if active {
			err := rc.ruleRunActions(rc.config.Actions, id)
			if err != nil {
				log.Println("Error running rule actions:", err)
			}

			err = rc.ruleInactiveActions(rc.config.ActionsInactive)
			if err != nil {
				log.Println("Error running rule inactive actions:", err)
			}
		} else {
			err := rc.ruleRunActions(rc.config.ActionsInactive, id)
			if err != nil {
				log.Println("Error running rule actions:", err)
			}

			err = rc.ruleInactiveActions(rc.config.Actions)
			if err != nil {
				log.Println("Error running rule inactive actions:", err)
			}
		}
	}

done:
	for {
		select {
		case <-rc.stop:
			break done
		case pts := <-rc.newRulePoints:
			// make sure the point is in a condition before we run the rule
			// otherwise, we can get into a loop
			found := false
			for _, c := range rc.config.Conditions {
				if c.ConditionType != data.PointValuePointValue {
					continue
				}
				if c.NodeID == pts.ID {
					found = true
					break
				}
			}

			if found {
				// found a condition that matches the point coming in, run the rule
				run(pts.ID, pts.Points)
			}

		case <-scheduleTicker.C:
			run(rc.config.ID, data.Points{{
				Time: time.Now(),
				Type: data.PointTypeTrigger,
			}})

		case <-deadlineTimer.C:
			// a pending period or a hold expired; re-evaluate with nothing
			// new to compare against
			run("", nil)

		case pts := <-rc.newPoints:
			err := data.MergePoints(pts.ID, pts.Points, &rc.config)
			if err != nil {
				log.Println("error merging rule points:", err)
			}

			if rc.hasSchedule() {
				scheduleTicker = time.NewTicker(scheduleTickTime)
			} else {
				scheduleTicker.Stop()
			}

			// send a schedule trigger through just in case someone changed a
			// schedule condition
			run("", data.Points{{
				Time: time.Now(),
				Type: data.PointTypeTrigger,
			}})

		case pts := <-rc.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &rc.config)
			if err != nil {
				log.Println("error merging rule edge points:", err)
			}

			run("", data.Points{{
				Time: time.Now(),
				Type: data.PointTypeTrigger,
			}})
		}
	}

	return rc.upSub.Unsubscribe()
}

// Stop sends a signal to the Run function to exit
func (rc *RuleClient) Stop(_ error) {
	close(rc.stop)
}

// Points is called by the Manager when new points for this
// node are received.
func (rc *RuleClient) Points(nodeID string, points []data.Point) {
	rc.newPoints <- NewPoints{nodeID, "", points}
}

// EdgePoints is called by the Manager when new edge points for this
// node are received.
func (rc *RuleClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	rc.newEdgePoints <- NewPoints{nodeID, parentID, points}
}

// sendPoint sets origin to the rule node
func (rc *RuleClient) sendPoint(id string, point data.Point) error {
	if id != rc.config.ID {
		// we must set origin as we are sending a point to something
		// other than the client root node
		// TODO: it might be good to somehow move this into the
		// client manager, so that clients don't need to worry about
		// setting Origin
		point.Origin = rc.config.ID
	}
	return SendNodePoint(rc.nc, id, point, false)
}

func (rc *RuleClient) hasSchedule() bool {
	for _, c := range rc.config.Conditions {
		if c.ConditionType == data.PointValueSchedule {
			return true
		}
	}
	return false
}

func (rc *RuleClient) processError(errS string) {
	if errS != "" {
		// always set rule error to the last error we encounter
		if errS != rc.config.Error {
			p := data.NewPointString(data.PointTypeError, "", errS)
			p.Time = time.Now()

			err := rc.sendPoint(rc.config.ID, p)
			if err != nil {
				log.Println("Rule error sending point:", err)
			} else {
				rc.config.Error = errS
			}
		}
	} else {
		// check if any other errors still exist
		found := ""

		for _, c := range rc.config.Conditions {
			if c.Error != "" {
				found = c.Error
				break
			}
		}

		for _, a := range rc.config.Actions {
			if a.Error != "" {
				found = a.Error
				break
			}
		}

		for _, a := range rc.config.ActionsInactive {
			if a.Error != "" {
				found = a.Error
				break
			}
		}

		if found != rc.config.Error {
			p := data.NewPointString(data.PointTypeError, "", found)
			p.Time = time.Now()

			err := rc.sendPoint(rc.config.ID, p)
			if err != nil {
				log.Println("Rule error sending point:", err)
			} else {
				rc.config.Error = found
			}
		}
	}
}

// condRuntime is the in-process state of a condition. The raw comparison
// result and the time it last changed are what the held-state pass debounces;
// the condition's active point is the held result and is what the rule, the
// UI, and history see.
//
// This state is not persisted. A pending period therefore restarts after a
// client restart or a configuration change that reloads the client, which is
// how Grafana behaves as well -- persisting a pending timer would mean writing
// a point on every evaluation.
type condRuntime struct {
	// raw is the last undebounced comparison result
	raw bool
	// rawChanged is when raw last took its current value
	rawChanged time.Time
	// deadline is when the held state changes on its own, zero when the
	// held state already agrees with raw
	deadline time.Time
}

// condRuntime returns the in-process state for a condition, seeding it from
// the condition's persisted active point the first time it is asked for.
func (rc *RuleClient) condRuntime(id string, active bool) *condRuntime {
	if rc.condState == nil {
		rc.condState = make(map[string]*condRuntime)
	}

	cs, ok := rc.condState[id]
	if !ok {
		cs = &condRuntime{raw: active, rawChanged: time.Now()}
		rc.condState[id] = cs
	}

	return cs
}

// ruleUpdateConditions compares incoming points against each condition and
// records the raw comparison result. The condition's active point is left to
// the held-state pass, which is the only place that decides what the rule
// sees.
// Currently, this function only processes the first point that matches -- this
// should handle all current uses.
func (rc *RuleClient) ruleUpdateConditions(nodeID string, points data.Points) {
	for _, p := range points {
		for i, c := range rc.config.Conditions {
			var active bool
			var errorActive bool

			processError := func(err error) {
				errorActive = true
				errS := err.Error()
				if c.Error != errS {
					p := data.NewPointString(data.PointTypeError, "", errS)
					p.Time = time.Now()

					log.Printf("Rule cond error %v:%v:%v\n", rc.config.Description, c.Description, err)
					err := rc.sendPoint(c.ID, p)
					if err != nil {
						log.Println("Rule error sending point:", err)
					} else {
						rc.config.Conditions[i].Error = errS
					}
				}
				rc.processError(errS)
			}

			switch c.ConditionType {
			case data.PointValuePointValue:
				if c.NodeID != "" && c.NodeID != nodeID {
					continue
				}

				if c.PointKey != "" && c.PointKey != p.Key {
					continue
				}

				if c.PointType != "" && c.PointType != p.Type {
					continue
				}
				// conditions match, so check value
				switch c.ValueType {
				case data.PointValueNumber:
					switch c.Operator {
					case data.PointValueGreaterThan:
						active = p.Val() > c.Value
					case data.PointValueLessThan:
						active = p.Val() < c.Value
					case data.PointValueEqual:
						active = p.Val() == c.Value
					case data.PointValueNotEqual:
						active = p.Val() != c.Value
					}
				case data.PointValueText:
					switch c.Operator {
					case data.PointValueEqual:
					case data.PointValueNotEqual:
					case data.PointValueContains:
					}
				case data.PointValueOnOff:
					condValue := c.Value != 0
					pointValue := p.Val() != 0
					active = condValue == pointValue
				default:
					processError(fmt.Errorf("unknown value type: %v", c.ValueType))
				}
			case data.PointValueSchedule:
				if p.Type != data.PointTypeTrigger {
					continue
				}

				weekdays := []time.Weekday{}
				for i, v := range c.Weekdays {
					if v {
						weekdays = append(weekdays, time.Weekday(i))
					}
				}
				sched := newSchedule(c.Start, c.End, weekdays, c.Dates)

				var err error
				active, err = sched.activeForTime(p.Time)
				if err != nil {
					processError(fmt.Errorf("error parsing schedule: %w", err))
					continue
				}
			}

			cs := rc.condRuntime(c.ID, c.Active)
			if active != cs.raw {
				cs.raw = active
				cs.rawChanged = time.Now()
				if nodeID != "" {
					// remember what moved the condition so an action that
					// runs later, when a pending period expires, can still
					// name the node that fired the rule
					rc.lastTrigger = nodeID
				}
			}

			if !errorActive && c.Error != "" {
				p := data.NewPointString(data.PointTypeError, "", "")
				p.Time = time.Now()

				err := rc.sendPoint(c.ID, p)
				if err != nil {
					log.Println("Rule error sending point:", err)
				} else {
					rc.config.Conditions[i].Error = ""
				}
				rc.processError("")
			}
		}
	}
}

// ruleApplyHeldState moves each condition's active point toward its raw state.
// It runs from a timer as well as from an inbound point, so it takes the
// current time rather than reading the clock per condition.
func (rc *RuleClient) ruleApplyHeldState(now time.Time) {
	live := make(map[string]bool, len(rc.config.Conditions))

	for i, c := range rc.config.Conditions {
		live[c.ID] = true

		cs := rc.condRuntime(c.ID, c.Active)
		cs.deadline = time.Time{}

		if cs.raw == c.Active {
			continue
		}

		if delay := c.holdTime(cs.raw); delay > 0 {
			deadline := cs.rawChanged.Add(delay)
			if now.Before(deadline) {
				cs.deadline = deadline
				continue
			}
		}

		rc.setConditionActive(i, cs.raw)
	}

	// drop state for conditions that are no longer children of the rule
	for id := range rc.condState {
		if !live[id] {
			delete(rc.condState, id)
		}
	}
}

// holdTime is how long a condition's raw result must hold before the
// condition's active point follows it. MinActive is the pending period: a
// spike that crosses the threshold and returns before it expires never
// activates the condition at all.
func (c Condition) holdTime(raw bool) time.Duration {
	if raw {
		return minutesToDuration(c.MinActive)
	}

	return 0
}

// minutesToDuration converts the fractional minutes the rule timing points are
// expressed in. Fractional values work, which is what lets the tests exercise
// the timing at hundredths of a minute.
func minutesToDuration(m float64) time.Duration {
	if m <= 0 {
		return 0
	}

	return time.Duration(m * float64(time.Minute))
}

// setConditionActive publishes a condition's active point
func (rc *RuleClient) setConditionActive(i int, active bool) {
	p := data.NewPointFloat(data.PointTypeActive, "", data.BoolToFloat(active))
	p.Time = time.Now()

	err := rc.sendPoint(rc.config.Conditions[i].ID, p)
	if err != nil {
		log.Println("Rule error sending point:", err)
	}

	rc.config.Conditions[i].Active = active
}

// ruleComputeActive computes the rule state from its conditions and publishes
// the rule's active point when it changes. A rule is active when every enabled
// condition is active and at least one condition is enabled.
func (rc *RuleClient) ruleComputeActive() bool {
	allActive := true
	activeConditionCount := 0

	for _, c := range rc.config.Conditions {
		if !c.Active && !c.Disabled {
			allActive = false
			break
		}
		if c.Active && !c.Disabled {
			activeConditionCount++
		}
	}

	if activeConditionCount == 0 && allActive {
		allActive = false
	}

	rc.setRuleActive(allActive)

	return allActive
}

// setRuleActive publishes the rule's active point when the state changes
func (rc *RuleClient) setRuleActive(active bool) {
	if active == rc.config.Active {
		return
	}

	p := data.NewPointFloat(data.PointTypeActive, "", data.BoolToFloat(active))
	p.Time = time.Now()

	err := rc.sendPoint(rc.config.ID, p)
	if err != nil {
		log.Println("Rule error sending point:", err)
	}

	rc.config.Active = active
}

// nextDeadline returns the earliest time at which the rule can change state on
// its own, with no inbound point -- a pending period expiring or a hold
// expiring. It returns the zero time when nothing is pending, in which case the
// client arms no timer at all.
func (rc *RuleClient) nextDeadline() time.Time {
	var next time.Time

	consider := func(t time.Time) {
		if t.IsZero() {
			return
		}
		if next.IsZero() || t.Before(next) {
			next = t
		}
	}

	for _, c := range rc.config.Conditions {
		if c.Disabled {
			continue
		}
		if cs, ok := rc.condState[c.ID]; ok {
			consider(cs.deadline)
		}
	}

	return next
}

// ruleRunActions runs rule actions
func (rc *RuleClient) ruleRunActions(actions []Action, triggerNodeID string) error {
	for i, a := range actions {
		if a.Disabled {
			continue
		}

		errorActive := false

		processError := func(err error) {
			errorActive = true
			errS := err.Error()
			if a.Error != errS {
				p := data.NewPointString(data.PointTypeError, "", errS)
				p.Time = time.Now()

				log.Printf("Rule action error %v:%v:%v\n", rc.config.Description, a.Description, err)
				err := rc.sendPoint(a.ID, p)
				if err != nil {
					log.Println("Rule error sending point:", err)
				} else {
					actions[i].Error = errS
				}
			}
			rc.processError(errS)
		}

		switch a.Action {
		case data.PointValueSetValue:
			if a.NodeID == "" {
				processError(fmt.Errorf("error, node action nodeID must be set"))
				break
			}

			if a.PointType == "" {
				processError(fmt.Errorf("error, node action point type must be set"))
				break
			}

			p := data.Point{
				Time:   time.Now(),
				Type:   a.PointType,
				Key:    a.PointKey,
				Origin: a.ID,
			}
			if a.ValueText != "" {
				p.PutString(a.ValueText)
			} else {
				p.PutFloat(a.Value)
			}

			err := rc.sendPoint(a.NodeID, p)
			if err != nil {
				log.Println("Error sending rule action point:", err)
			}
		case data.PointValueNotify:
			// get node that fired the rule; "all" asks for every living
			// instance of the node, which is how a node is fetched when
			// the caller knows the ID but not the parent
			nodes, err := GetNodes(rc.nc, "all", triggerNodeID, "", false)
			if err != nil {
				processError(err)
				break
			}

			if len(nodes) < 1 {
				processError(fmt.Errorf("trigger node not found"))
				break
			}

			triggerNode := nodes[0]

			triggerNodeDesc := triggerNode.Desc()

			n := data.Notification{
				ID:         uuid.New().String(),
				SourceNode: triggerNodeID,
				Subject:    rc.config.Description,
				Message:    rc.config.Description + " fired at " + triggerNodeDesc,
			}

			p, err := n.Point()
			if err != nil {
				processError(fmt.Errorf("error encoding notification: %w", err))
				break
			}

			err = rc.sendPoint(rc.config.ID, p)
			if err != nil {
				processError(fmt.Errorf("error sending notification point: %w", err))
			}
		case data.PointValuePlayAudio:
			f, err := os.Open(a.FilePath)
			if err != nil {
				processError(fmt.Errorf("error opening wave file: %w", err))
				break
			}

			d := wav.NewDecoder(f)
			d.ReadInfo()

			format := d.Format()

			if err := f.Close(); err != nil {
				log.Println("Rule action: error closing wave file:", err)
			}

			if format.SampleRate < 8000 {
				processError(fmt.Errorf("invalid wave file sample rate: %v", format.SampleRate))
				break
			}

			channelNum := strconv.Itoa(a.Channel)
			sampleRate := strconv.Itoa(format.SampleRate)

			go func() {
				stderr, err := exec.Command("speaker-test", "-D"+a.Device, "-twav", "-w"+a.FilePath, "-c5", "-s"+channelNum, "-r"+sampleRate).CombinedOutput()
				if err != nil {
					log.Println("Play audio error:", err)
					log.Printf("Audio stderr: %s\n", stderr)
				}
			}()
		default:
			processError(fmt.Errorf("unknown rule action: %v", a.Action))
		}

		p := data.NewPointFloat(data.PointTypeActive, "", 1)
		err := rc.sendPoint(a.ID, p)
		if err != nil {
			log.Println("Error sending rule action point:", err)
		}

		actions[i].Active = true

		if !errorActive && a.Error != "" {
			p := data.NewPointString(data.PointTypeError, "", "")
			p.Time = time.Now()

			err := rc.sendPoint(a.ID, p)
			if err != nil {
				log.Println("Rule error sending point:", err)
			} else {
				actions[i].Error = ""
			}
			rc.processError("")
		}

	}
	return nil
}

func (rc *RuleClient) ruleInactiveActions(actions []Action) error {
	for i, a := range actions {
		if a.Disabled {
			continue
		}

		p := data.NewPointFloat(data.PointTypeActive, "", 0)
		err := rc.sendPoint(a.ID, p)
		if err != nil {
			log.Println("Error sending rule action point:", err)
		}
		actions[i].Active = false
	}
	return nil
}
