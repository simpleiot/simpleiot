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

// RuleTickerMinDuration is the minimum duration for the rule ticker. The ticker
// is used to evaluate conditions with sample windows or schedules.
const RuleTickerMinDuration = 10 * time.Second

// Rule represent a rule node config
type Rule struct {
	ID              string      `node:"id"`
	Parent          string      `node:"parent"`
	Description     string      `point:"description"`
	Disabled        bool        `point:"disabled"`
	Active          bool        `point:"active"`
	Error           string      `point:"error"`
	ActionCooldown  float64     `point:"actionCooldown"` // in seconds
	Conditions      []Condition `child:"condition"`
	Actions         []Action    `child:"action"`
	ActionsInactive []Action    `child:"actionInactive"`
}

func (r Rule) String() string {
	ret := fmt.Sprintf("Rule: %v\n", r.Description)
	ret += fmt.Sprintf("  active: %v\n", r.Active)
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
	/*** General condition information */
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	// ConditionType is "pointValue" or "schedule"
	ConditionType string `point:"conditionType"`

	// MinActive is currently unused
	// MinActive float64 `point:"minActive"`

	// Active indicates whether or not the condition is active
	Active bool `point:"active"`
	// Error contains an error message describing the condition's error state
	Error string `point:"error"`

	/*** Options for pointValue conditions */
	// NodeID, PointType, and PointKey filter the points that are relevant for
	// this condition.
	NodeID    string `point:"nodeID"`
	PointType string `point:"pointType"`
	PointKey  string `point:"pointKey"`
	// ValueType is one of: "onOff", "number", or "text"
	ValueType string `point:"valueType"`
	// Value contains the boolean or numeric value to compare with the incoming
	// point values
	Value float64 `point:"value"`
	// ValueText contains the string to compare with the incoming point values
	ValueText string `point:"valueText"`
	// Operator is used to compare Value and ValueText against incoming point
	// values to determine if the condition is active or not.
	//
	// - For ValueType of "onOff", operator is ignored
	// - For ValueType of "number", operator can be ">", "<", "=", or "!="
	// - For ValueType of "text", operator can be "=", "!+", or "contains"
	Operator string `point:"operator"`
	// Window specifies a duration (in seconds) where a WindowPercent percentage
	// of samples within the window must pass the condition in order for the
	// entire condition to be active. Only works when ValueType is "onOff" or
	// "number".
	//
	// For this to work correctly, an Influx DB node must be a child of any
	// ancestor of this condition node. The condition will query Influx
	// approximately at the end of each window to determine if the condition
	// should be active or not.
	Window        float64 `point:"window"`
	WindowPercent float64 `point:"windowPercent"`
	// The window is evaluated every WindowEvaluate seconds. This value may not
	// be less than RuleTickerMinDuration.
	WindowEvaluate float64 `point:"windowEvaluate"`

	/*** Options for schedule conditions */
	// Start and End indicate the time of day to activate the condition
	Start string `point:"start"`
	End   string `point:"end"`
	// Weekdays and Dates controls which weekdays or custom dates the condition
	// should activate
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
		ret = fmt.Sprintf("  COND: %v  CTYPE:%v  VTYPE:%v  OP:%v  V:%v",
			c.Description, c.ConditionType, c.ValueType, c.Operator, value)
		if c.NodeID != "" {
			ret += fmt.Sprintf("  NODEID:%v", c.NodeID)
		}
		if c.PointType != "" {
			ret += fmt.Sprintf("  PTYPE:%v", c.PointType)
		}
		if c.PointKey != "" {
			ret += fmt.Sprintf("  PKEY:%v", c.PointKey)
		}
		// if c.MinActive > 0 {
		// 	ret += fmt.Sprintf("  MINACT:%v", c.MinActive)
		// }
		if c.HasWindow() {
			ret += fmt.Sprintf(
				"  WINDOW:%v, %v%% every %v",
				time.Duration(c.Window)*time.Second, c.WindowPercent,
				time.Duration(c.WindowEvaluate)*time.Second,
			)
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

// HasWindow returns true if and only if the condition uses sample windowing
func (c Condition) HasWindow() bool {
	return c.ConditionType == data.PointValuePointValue &&
		c.Window > 0
}

// Evaluate returns true if and only if the specified slice of Points should
// activate the condition
// func (c *Condition) Evaluate(nodeID string, points data.Points) bool {
// 	for _, p := range points {

// 	}
// }

// Action defines actions that can be taken if a rule is active.
type Action struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	Active      bool   `point:"active"`
	Error       string `point:"error"`
	// Action: notify, setValue, playAudio
	Action    string `point:"action"`
	NodeID    string `point:"nodeID"`
	PointType string `point:"pointType"`
	// PointType: number, text, onOff
	ValueType string  `point:"valueType"`
	Value     float64 `point:"value"`
	ValueText string  `point:"valueText"`
	// the following are used for audio playback
	PointChannel  int    `point:"pointChannel"`
	PointDevice   string `point:"pointDevice"`
	PointFilePath string `point:"pointFilePath"`
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
	ret := fmt.Sprintf("%v  ACT:%v  VTYPE:%v  V:%v",
		a.Description, a.Action, a.ValueType, value)
	if a.NodeID != "" {
		ret += fmt.Sprintf("  NODEID:%v", a.NodeID)
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
	// PointType: number, text, onOff
	ValueType string  `point:"valueType"`
	Value     float64 `point:"value"`
	ValueText string  `point:"valueText"`
	// the following are used for audio playback
	PointChannel  int    `point:"pointChannel"`
	PointDevice   string `point:"pointDevice"`
	PointFilePath string `point:"pointFilePath"`
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
	dbNodeID      string
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
	// watch all points that flow through parent node
	// TODO: we should optimize this so we only watch the nodes
	// that are in the conditions
	subject := fmt.Sprintf("up.%v.*", rc.config.Parent)

	var err error
	rc.upSub, err = rc.nc.Subscribe(subject, func(msg *nats.Msg) {
		points, err := data.PbDecodePoints(msg.Data)
		if err != nil {
			log.Println("Error decoding points in rule upSub:", err)
			return
		}

		// find node ID for points
		chunks := strings.Split(msg.Subject, ".")
		if len(chunks) != 3 {
			log.Println("rule client up sub, malformed subject:", msg.Subject)
			return
		}

		rc.newRulePoints <- NewPoints{chunks[2], "", points}
	})

	if err != nil {
		return fmt.Errorf("Rule error subscribing to upsub: %v", err)
	}

	ticker := time.NewTicker(RuleTickerMinDuration)
	updateClient := func() {
		// TODO: ticker is a brute force way to do this; we could optimize at
		// some point by creating a timer to expire on the next schedule change
		// or when the sample window ends
		var tickerDuration time.Duration
		var hasWindowCond bool
		for _, c := range rc.config.Conditions {
			if c.ConditionType == data.PointValueSchedule {
				tickerDuration = RuleTickerMinDuration
				break // already set to minimum tickerDuration
			}
			if c.HasWindow() {
				windowDur := time.Duration(c.Window) * time.Second
				// Use condition window as tick interval as long as it's larger
				// than RuleTickerMinDuration
				if windowDur < RuleTickerMinDuration {
					tickerDuration = RuleTickerMinDuration
					break // already set to minimum tickerDuration
				} else if tickerDuration == 0 || windowDur < tickerDuration {
					// set ticker duration to condition window
					tickerDuration = windowDur
				}
			}
		}

		// Update ticker
		if tickerDuration > 0 {
			ticker.Reset(tickerDuration)
		} else {
			ticker.Stop()
		}

		// Update dbNodeID
		if !hasWindowCond {
			rc.dbNodeID = "" // We don't need to keep track of this
		} else {
			// If we have a condition that uses a sample window, we need to
			// locate the closest Influx DB node
			dbNode, err := rc.findDbNodes(rc.config.Parent)
			if err != nil {
				rc.processError(fmt.Sprintf(
					"Condition uses Window: Error finding Db node: %s",
					err,
				))
			} else {
				log.Printf("rule %s: Located Db node: %s", rc.config.ID, dbNode.ID)
				rc.dbNodeID = dbNode.ID
			}
		}
	}

	run := func(id string, pts data.Points) {
		var active, changed bool
		var err error

		if len(pts) == 0 {
			pts = data.Points{{
				Time: time.Now(),
				Type: data.PointTypeTrigger,
			}}
		}
		active, changed, err = rc.ruleProcessPoints(id, pts)
		if err != nil {
			log.Println("Error processing rule point:", err)
		}

		if !changed {
			return
		}

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

	updateClient()
done:
	for {
		select {
		case <-rc.stop:
			break done
		case pts := <-rc.newRulePoints:
			run(pts.ID, pts.Points)

		case <-ticker.C:
			run(rc.config.ID, nil)

		case pts := <-rc.newPoints:
			err := data.MergePoints(pts.ID, pts.Points, &rc.config)
			if err != nil {
				log.Println("error merging rule points:", err)
			}
			updateClient()
			// Run in case schedule condition has changed
			run(rc.config.ID, nil)
		case pts := <-rc.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &rc.config)
			if err != nil {
				log.Println("error merging rule edge points:", err)
			}
			updateClient()
			// Run in case schedule condition has changed
			run(rc.config.ID, nil)
		}
	}

	// Clean up
	ticker.Stop()
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

// find the closest Db node by searching the children of nodeID and its
// ancestors
func (rc *RuleClient) findDbNodes(nodeID string) (*Db, error) {
	// Fetch children for nodeID to find a Db node
	dbNodes, err := GetNodesType[Db](rc.nc, nodeID, "all")
	if err != nil {
		return nil, err
	}

	// Return first found Db node
	if len(dbNodes) > 0 {
		return &dbNodes[0], nil
	}

	// Recursively search ancestor nodes
	ne, err := GetNodes(rc.nc, "all", nodeID, "", false)
	if err != nil {
		return nil, err
	}
	for _, nodeEdge := range ne {
		dbNodeEdge, err := rc.findDbNodes(nodeEdge.ID)
		if err != nil {
			return nil, err
		}
		if dbNodeEdge != nil {
			return dbNodeEdge, nil
		}
	}

	// None found
	return nil, nil
}

func (rc *RuleClient) processError(errS string) {
	if errS != "" {
		// always set rule error to the last error we encounter
		if errS != rc.config.Error {
			p := data.Point{
				Type: data.PointTypeError,
				Time: time.Now(),
				Text: errS,
			}

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
			p := data.Point{
				Type: data.PointTypeError,
				Time: time.Now(),
				Text: found,
			}

			err := rc.sendPoint(rc.config.ID, p)
			if err != nil {
				log.Println("Rule error sending point:", err)
			} else {
				rc.config.Error = found
			}
		}
	}
}

// processConditionError updates the Error point on the condition at the
// specified index if it differs from err; then updates the rule Error by
// calling processError
func (rc *RuleClient) processConditionError(condIndex int, err error) {
	c := &rc.config.Conditions[condIndex]
	errS := err.Error()
	if c.Error == errS {
		return
	}

	p := data.Point{
		Type: data.PointTypeError,
		Time: time.Now(),
		Text: errS,
	}

	log.Printf(
		"Rule condition error %v:%v:%v\n",
		rc.config.Description, c.Description, err,
	)
	err = rc.sendPoint(c.ID, p)
	if err != nil {
		log.Println("Rule error sending point:", err)
	} else {
		c.Error = errS
	}
	// set error on rule itself
	// Note: no need to do this unless error has changed
	rc.processError(errS)
}

// ruleProcessPoints runs points through a rules conditions and and updates condition
// and rule active status. Returns true if point was processed and active is true.
// Currently, this function only processes the first point that matches -- this should
// handle all current uses.
func (rc *RuleClient) ruleProcessPoints(nodeID string, points data.Points) (bool, bool, error) {
	for _, p := range points {
		for i, c := range rc.config.Conditions {
			var active bool
			var errorActive bool

			processError := func(err error) {
				errorActive = true
				errS := err.Error()
				if c.Error != errS {
					p := data.Point{
						Type: data.PointTypeError,
						Time: time.Now(),
						Text: errS,
					}

					log.Printf(
						"Rule cond error %v:%v:%v\n",
						rc.config.Description, c.Description, err,
					)
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

				// if c.HasWindow() {
				// 	if p.Type != data.PointTypeTrigger {
				// 		// Conditions that require a sample window should not respond
				// 		// to inbound points, only triggers from the ticker
				// 		continue
				// 	}
				// 	// Query Influx DB node
				// 	query := data.HistoryQuery{
				// 		Start:           startTime,
				// 		Stop:            stopTime,
				// 		TagFilters:      nil,
				// 		AggregateWindow: data.Window,
				// 	}
				// 	data, err := json.Marshal(query)
				// 	if err != nil {
				// 		processError(fmt.Errorf(
				// 			"query db: encode query: %w", err,
				// 		))
				// 		break condTypeSwitch

				// 	}
				// 	rc.nc.Request("history." + rc.dbNodeID, data, ...)
				// 	break condTypeSwitch
				// }

				// conditions match, so check value
				switch c.ValueType {
				case data.PointValueNumber:
					switch c.Operator {
					case data.PointValueGreaterThan:
						active = p.Value > c.Value
					case data.PointValueLessThan:
						active = p.Value < c.Value
					case data.PointValueEqual:
						active = p.Value == c.Value
					case data.PointValueNotEqual:
						active = p.Value != c.Value
					}
				case data.PointValueText:
					switch c.Operator {
					case data.PointValueEqual:
						active = p.Text == c.ValueText
					case data.PointValueNotEqual:
						active = p.Text != c.ValueText
					case data.PointValueContains:
						active = strings.Contains(p.Text, c.ValueText)
					}
				case data.PointValueOnOff:
					condValue := c.Value != 0
					pointValue := p.Value != 0
					active = condValue == pointValue
				default:
					processError(fmt.Errorf("unknown value type: %v", c.ValueType))
					continue
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
					processError(fmt.Errorf("Error parsing schedule: %w", err))
					continue
				}
			}

			if active != c.Active {
				// update condition
				p := data.Point{
					Type:  data.PointTypeActive,
					Time:  time.Now(),
					Value: data.BoolToFloat(active),
				}

				err := rc.sendPoint(c.ID, p)
				if err != nil {
					log.Println("Rule error sending point:", err)
				}

				rc.config.Conditions[i].Active = active
			}

			if !errorActive && c.Error != "" {
				p := data.Point{
					Type: data.PointTypeError,
					Time: time.Now(),
					Text: "",
				}

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

	allActive := true

	for _, c := range rc.config.Conditions {
		if !c.Active {
			allActive = false
			break
		}
	}

	changed := false

	if allActive != rc.config.Active {
		p := data.Point{
			Type:  data.PointTypeActive,
			Time:  time.Now(),
			Value: data.BoolToFloat(allActive),
		}

		err := rc.sendPoint(rc.config.ID, p)
		if err != nil {
			log.Println("Rule error sending point:", err)
		}
		changed = true

		rc.config.Active = allActive
	}

	return allActive, changed, nil
}

// ruleRunActions runs rule actions
func (rc *RuleClient) ruleRunActions(actions []Action, triggerNodeID string) error {
	for i, a := range actions {
		errorActive := false

		processError := func(err error) {
			errorActive = true
			errS := err.Error()
			if a.Error != errS {
				p := data.Point{
					Type: data.PointTypeError,
					Time: time.Now(),
					Text: errS,
				}

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
				processError(fmt.Errorf("Error, node action nodeID must be set"))
				break
			}

			if a.PointType == "" {
				processError(fmt.Errorf("Error, node action point type must be set"))
				break
			}

			p := data.Point{
				Time:   time.Now(),
				Type:   a.PointType,
				Value:  a.Value,
				Text:   a.ValueText,
				Origin: a.ID,
			}
			err := rc.sendPoint(a.NodeID, p)
			if err != nil {
				log.Println("Error sending rule action point:", err)
			}
		case data.PointValueNotify:
			// get node that fired the rule
			nodes, err := GetNodes(rc.nc, "none", triggerNodeID, "", false)
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
				SourceNode: a.NodeID,
				Message:    rc.config.Description + " fired at " + triggerNodeDesc,
			}

			// TODO this notify code needs to be reworked
			d, err := n.ToPb()

			if err != nil {
				return err
			}

			err = rc.nc.Publish("node."+rc.config.ID+".not", d)

			if err != nil {
				return err
			}
		case data.PointValuePlayAudio:
			f, err := os.Open(a.PointFilePath)
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()

			d := wav.NewDecoder(f)
			d.ReadInfo()

			format := d.Format()

			if format.SampleRate < 8000 {
				log.Println("Rule action: invalid wave file sample rate:", format.SampleRate)
				continue
			}

			channelNum := strconv.Itoa(a.PointChannel)
			sampleRate := strconv.Itoa(format.SampleRate)

			go func() {
				stderr, err := exec.Command("speaker-test", "-D"+a.PointDevice, "-twav", "-w"+a.PointFilePath, "-c5", "-s"+channelNum, "-r"+sampleRate).CombinedOutput()
				if err != nil {
					log.Println("Play audio error:", err)
					log.Printf("Audio stderr: %s\n", stderr)
				}
			}()
		default:
			processError(fmt.Errorf("Unknown rule action: %v", a.Action))
		}

		p := data.Point{
			Type:  data.PointTypeActive,
			Value: 1,
		}
		err := rc.sendPoint(a.ID, p)
		if err != nil {
			log.Println("Error sending rule action point:", err)
		}

		actions[i].Active = true

		if !errorActive && a.Error != "" {
			p := data.Point{
				Type: data.PointTypeError,
				Time: time.Now(),
				Text: "",
			}

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
		p := data.Point{
			Type:  data.PointTypeActive,
			Value: 0,
		}
		err := rc.sendPoint(a.ID, p)
		if err != nil {
			log.Println("Error sending rule action point:", err)
		}
		actions[i].Active = false
	}
	return nil
}
