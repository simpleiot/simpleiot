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

	// used with shedule rules
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

	// TODO schedule ticker is a brute force way to do this
	// we could optimize at some point by creating a timer to expire
	// on the next schedule change
	scheduleTickTime := time.Second * 10
	scheduleTicker := time.NewTicker(scheduleTickTime)
	if !rc.hasSchedule() {
		scheduleTicker.Stop()
	}

	run := func(id string, pts data.Points) {
		var active, changed bool
		var err error

		if rc.config.Disabled {
			active = false
		} else {
			if len(pts) > 0 {
				active, changed, err = rc.ruleProcessPoints(id, pts)
				if err != nil {
					log.Println("Error processing rule point:", err)
				}

				if !changed {
					return
				}
			} else {
				// send a schedule trigger through just in case someone changed a
				// schedule condition
				active, _, err = rc.ruleProcessPoints(rc.config.ID, data.Points{{
					Time: time.Now(),
					Type: data.PointTypeTrigger,
				}})

				if err != nil {
					log.Println("Error processing rule point:", err)
				}
			}
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
			run("", nil)

		case pts := <-rc.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &rc.config)
			if err != nil {
				log.Println("error merging rule edge points:", err)
			}
			run("", nil)
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
					case data.PointValueNotEqual:
					case data.PointValueContains:
					}
				case data.PointValueOnOff:
					condValue := c.Value != 0
					pointValue := p.Value != 0
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
		if a.Disabled {
			continue
		}

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
				Key:    a.PointKey,
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
			processError(fmt.Errorf("Uknown rule action: %v", a.Action))
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
		if a.Disabled {
			continue
		}

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
