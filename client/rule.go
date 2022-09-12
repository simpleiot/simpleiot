package client

import (
	"fmt"
	"log"
	"time"

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
	MinTimeActive float64 `point:"minTimeActive"`
	Active        bool    `point:"active"`

	// used with point value rules
	NodeID         string  `point:"nodeID"`
	PointType      string  `point:"pointType"`
	PointKey       string  `point:"pointKey"`
	PointIndex     int     `point:"pointIndex"`
	PointValueType string  `point:"pointValueType"`
	Operator       string  `point:"operator"`
	PointValue     float64 `point:"pointValue"`
	PointTextValue string  `point:"pointTextValue"`

	// used with shedule rules
	StartTime string `point:"startTime"`
	EndTime   string `point:"endTime"`
	Weekdays  []time.Weekday
}

func (c Condition) String() string {
	ret := fmt.Sprintf("  COND: %v, V:%v, A:%v\n", c.Description, c.PointValue, c.Active)
	return ret
}

// Action defines actions that can be taken if a rule is active.
type Action struct {
	ID             string  `node:"id"`
	Parent         string  `node:"parent"`
	Description    string  `point:"description"`
	Action         string  `point:"action"`
	NodeID         string  `point:"nodeID"`
	PointType      string  `point:"pointType"`
	PointValueType string  `point:"pointValueType"`
	PointValue     float64 `point:"pointValue"`
	PointTextValue string  `point:"pointTextValue"`
	PointChannel   int     `point:"pointChannel"`
	PointDevice    string  `point:"pointDevice"`
	PointFilePath  string  `point:"pointFilePath"`
}

func (a Action) String() string {
	ret := fmt.Sprintf("%v, %v\n", a.Description, a.PointValue)
	return ret
}

// ActionInactive defines actions that can be taken if a rule is inactive.
// this is defined for use with the client.SendNodeType API
type ActionInactive struct {
	ID             string  `node:"id"`
	Parent         string  `node:"parent"`
	Description    string  `point:"description"`
	Action         string  `point:"action"`
	NodeID         string  `point:"nodeID"`
	PointType      string  `point:"pointType"`
	PointValueType string  `point:"pointValueType"`
	PointValue     float64 `point:"pointValue"`
	PointTextValue string  `point:"pointTextValue"`
	PointChannel   int     `point:"pointChannel"`
	PointDevice    string  `point:"pointDevice"`
	PointFilePath  string  `point:"pointFilePath"`
}

// RuleClient is a SIOT client used to run rules
type RuleClient struct {
	nc            *nats.Conn
	config        Rule
	stop          chan struct{}
	newPoints     chan []data.Point
	newEdgePoints chan []data.Point
}

// NewRuleClient ...
func NewRuleClient(nc *nats.Conn, config Rule) Client {
	return &RuleClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan []data.Point),
		newEdgePoints: make(chan []data.Point),
	}
}

// Start runs the main logic for this client and blocks until stopped
func (rc *RuleClient) Start() error {
	for {
		select {
		case <-rc.stop:
			return nil
		case pts := <-rc.newPoints:
			err := data.MergePoints(pts, &rc.config)
			if err != nil {
				log.Println("error merging rule points: ", err)
			}
		case pts := <-rc.newEdgePoints:
			err := data.MergeEdgePoints(pts, &rc.config)
			if err != nil {
				log.Println("error merging rule edge points: ", err)
			}
		}
	}
}

// Stop sends a signal to the Start function to exit
func (rc *RuleClient) Stop(err error) {
	close(rc.stop)
}

// Points is called by the Manager when new points for this
// node are received.
func (rc *RuleClient) Points(nodeID string, points []data.Point) {
	rc.newPoints <- points
}

// EdgePoints is called by the Manager when new edge points for this
// node are received.
func (rc *RuleClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	rc.newEdgePoints <- points
}
