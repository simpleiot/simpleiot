package client

import (
	"errors"
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
	Disable         bool        `point:"disable"`
	Active          bool        `point:"active"`
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
	// general parameters
	ID            string  `node:"id"`
	Parent        string  `node:"parent"`
	Description   string  `point:"description"`
	ConditionType string  `point:"conditionType"`
	MinActive     float64 `point:"minActive"`
	Active        bool    `point:"active"`

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
	StartTime string `point:"startTime"`
	EndTime   string `point:"endTime"`
	Weekdays  []time.Weekday
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

	ret := fmt.Sprintf("  COND: %v  CTYPE:%v  VTYPE:%v  V:%v",
		c.Description, c.ConditionType, c.ValueType, value)
	if c.NodeID != "" {
		ret += fmt.Sprintf("  NODEID:%v", c.NodeID)
	}
	if c.MinActive > 0 {
		ret += fmt.Sprintf("  MINACT:%v", c.MinActive)
	}
	ret += fmt.Sprintf("  A:%v", c.Active)
	ret += "\n"
	return ret
}

// Action defines actions that can be taken if a rule is active.
type Action struct {
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
}

// NewRuleClient ...
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

// Start runs the main logic for this client and blocks until stopped
func (rc *RuleClient) Start() error {
	// watch all points that flow through parent node
	// FIXME: we should optimize this so we only watch the nodes
	// that are in the conditions
	subject := fmt.Sprintf("up.%v.*.points", rc.config.Parent)

	var err error
	rc.upSub, err = rc.nc.Subscribe(subject, func(msg *nats.Msg) {
		points, err := data.PbDecodePoints(msg.Data)
		if err != nil {
			log.Println("Error decoding points in rule upSub: ", err)
			return
		}

		// find node ID for points
		chunks := strings.Split(msg.Subject, ".")
		if len(chunks) != 4 {
			log.Println("rule client up sub, malformed subject: ", msg.Subject)
			return
		}

		rc.newRulePoints <- NewPoints{chunks[2], "", points}
	})

	if err != nil {
		return fmt.Errorf("Rule error subscribing to upsub: %v", err)
	}

done:
	for {
		select {
		case <-rc.stop:
			break done
		case pts := <-rc.newRulePoints:
			active, changed, err := rc.ruleProcessPoints(pts.ID, pts.Points)

			if err != nil {
				log.Println("Error processing rule point: ", err)
			}

			if !changed {
				continue
			}

			if active {
				err := rc.ruleRunActions(rc.config.Actions, pts.ID)
				if err != nil {
					log.Println("Error running rule actions: ", err)
				}

				err = rc.ruleRunInactiveActions(rc.config.ActionsInactive)
				if err != nil {
					log.Println("Error running rule inactive actions: ", err)
				}
			} else {
				err := rc.ruleRunActions(rc.config.ActionsInactive, pts.ID)
				if err != nil {
					log.Println("Error running rule actions: ", err)
				}

				err = rc.ruleRunInactiveActions(rc.config.Actions)
				if err != nil {
					log.Println("Error running rule inactive actions: ", err)
				}
			}

		case pts := <-rc.newPoints:
			err := data.MergePoints(pts.ID, pts.Points, &rc.config)
			if err != nil {
				log.Println("error merging rule points: ", err)
			}
		case pts := <-rc.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &rc.config)
			if err != nil {
				log.Println("error merging rule edge points: ", err)
			}
		}
	}

	rc.upSub.Unsubscribe()

	return nil
}

// Stop sends a signal to the Start function to exit
func (rc *RuleClient) Stop(err error) {
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
	return SendNodePoint(rc.nc, id, point, false)
}

// ruleProcessPoints runs points through a rules conditions and and updates condition
// and rule active status. Returns true if point was processed and active is true.
// Currently, this function only processes the first point that matches -- this should
// handle all current uses.
func (rc *RuleClient) ruleProcessPoints(nodeID string, points data.Points) (bool, bool, error) {
	pointsProcessed := false

	for _, p := range points {
		for i, c := range rc.config.Conditions {
			var active bool

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
					pointsProcessed = true
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
					pointsProcessed = true
					switch c.Operator {
					case data.PointValueEqual:
					case data.PointValueNotEqual:
					case data.PointValueContains:
					}
				case data.PointValueOnOff:
					pointsProcessed = true
					condValue := c.Value != 0
					pointValue := p.Value != 0
					active = condValue == pointValue
				default:
					log.Printf("unknown point type for rule: %v: %v\n",
						rc.config.Description, c.ValueType)
				}
			case data.PointValueSchedule:
				if p.Type != data.PointTypeTrigger {
					continue
				}
				pointsProcessed = true
				sched := newSchedule(c.StartTime, c.EndTime, c.Weekdays)

				var err error
				active, err = sched.activeForTime(p.Time)
				if err != nil {
					log.Println("Error parsing schedule time: ", err)
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
					log.Println("Rule error sending point: ", err)
				}

				rc.config.Conditions[i].Active = active
			}
		}
	}

	if pointsProcessed {
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
				log.Println("Rule error sending point: ", err)
			}
			changed = true

			rc.config.Active = allActive
		}

		return allActive, changed, nil
	}

	return false, false, nil
}

// ruleRunActions runs rule actions
func (rc *RuleClient) ruleRunActions(actions []Action, triggerNodeID string) error {
	for i, a := range actions {
		switch a.Action {
		case data.PointValueSetValue:
			if a.NodeID == "" {
				log.Println("Error, node action nodeID must be set, action id: ", a.ID)
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
				log.Println("Error sending rule action point: ", err)
			}
		case data.PointValueNotify:
			// get node that fired the rule
			nodes, err := GetNodes(rc.nc, "none", triggerNodeID, "", false)
			if err != nil {
				return err
			}

			if len(nodes) < 1 {
				return errors.New("trigger node not found")
			}

			triggerNode := nodes[0]

			triggerNodeDesc := triggerNode.Desc()

			n := data.Notification{
				ID:         uuid.New().String(),
				SourceNode: a.NodeID,
				Message:    rc.config.Description + " fired at " + triggerNodeDesc,
			}

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
				log.Println("Rule action: invalid wave file sample rate: ", format.SampleRate)
				continue
			}

			channelNum := strconv.Itoa(a.PointChannel)
			sampleRate := strconv.Itoa(format.SampleRate)

			go func() {
				stderr, err := exec.Command("speaker-test", "-D"+a.PointDevice, "-twav", "-w"+a.PointFilePath, "-c5", "-s"+channelNum, "-r"+sampleRate).CombinedOutput()
				if err != nil {
					log.Println("Play audio error: ", err)
					log.Printf("Audio stderr: %s\n", stderr)
				}
			}()
		default:
			log.Println("Uknown rule action: ", a.Action)
		}

		p := data.Point{
			Type:  data.PointTypeActive,
			Value: 1,
		}
		err := rc.sendPoint(a.ID, p)
		if err != nil {
			log.Println("Error sending rule action point: ", err)
		}

		actions[i].Active = true
	}
	return nil
}

func (rc *RuleClient) ruleRunInactiveActions(actions []Action) error {
	for i, a := range actions {
		p := data.Point{
			Type:  data.PointTypeActive,
			Value: 0,
		}
		err := rc.sendPoint(a.ID, p)
		if err != nil {
			log.Println("Error sending rule action point: ", err)
		}
		actions[i].Active = false
	}
	return nil
}

/* FIXME -- this should be moved to rules client
childNodes, err := st.db.nodeDescendents(st.db.rootNodeID(), "", false, false)
if err != nil {
	log.Println("Error getting child nodes to run schedule: ", err)
} else {
	for _, c := range childNodes {
		err := st.runSchedule(c)
		if err != nil {
			log.Println("Error running schedule: ", err)
		}
	}
}
t.Reset(time.Second * 5)
*/
