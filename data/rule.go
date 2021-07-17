package data

import (
	"fmt"
	"time"
)

// Condition defines parameters to look for in a sample. Either SampleType or SampleID
// (or both) can be set. They can't both be "".
type Condition struct {
	ID             string
	Description    string
	NodeID         string
	PointType      string
	PointID        string
	PointIndex     int
	PointValueType string
	Operator       string
	PointValue     float64
	PointTextValue string
	MinTimeActive  float64
	Active         bool
}

// Action defines actions that can be taken if a rule is active.
// Template can optionally be used to customize the message that is sent and
// uses Io Type or IDs to fill in the values. Example might be:
// JamMonitoring: Alert: {{ description }} is in ALARM state with tank level of {{ tankLevel }}.
type Action struct {
	ID             string
	Description    string
	Action         string
	NodeID         string
	PointType      string
	PointValueType string
	PointValue     float64
	PointTextValue string
	PointFilePath  string
}

// RuleConfig contains parts of the rule that the user changes
type RuleConfig struct {
}

// RuleState contains parts of a rule that the system changes
type RuleState struct {
	Active     bool      `json:"active"`
	LastAction time.Time `json:"lastAction"`
}

// Rule defines a conditions and actions that are run if condition is true. Global indicates
// the rule applies to all Devices. The rule config and state is separated so we can make updates
// to the Rule without config affecting state, and state affecting config as these are typically
// done by two different entities.
type Rule struct {
	ID          string
	Description string
	Active      bool
	Conditions  []Condition
	Actions     []Action
}

// NodeToRule converts nodes that make up a rule to a node
func NodeToRule(ruleNode NodeEdge, conditionNodes, actionNodes []NodeEdge) (*Rule, error) {
	ret := &Rule{}
	ret.ID = ruleNode.ID
	for _, p := range ruleNode.Points {
		switch p.Type {
		case PointTypeDescription:
			ret.Description = p.Text
		case PointTypeActive:
			ret.Active = FloatToBool(p.Value)
		}
	}

	for _, cond := range conditionNodes {
		var newCond Condition
		newCond.ID = cond.ID
		newCond.PointIndex = -1
		for _, p := range cond.Points {
			switch p.Type {
			case PointTypeDescription:
				newCond.Description = p.Text
			case PointTypeID:
				newCond.NodeID = p.Text
			case PointTypePointType:
				newCond.PointType = p.Text
			case PointTypePointID:
				newCond.PointID = p.Text
			case PointTypePointIndex:
				newCond.PointIndex = int(p.Value)
			case PointTypeValueType:
				newCond.PointValueType = p.Text
			case PointTypeOperator:
				newCond.Operator = p.Text
			case PointTypeValue:
				newCond.PointValue = p.Value
			case PointTypeMinActive:
				newCond.MinTimeActive = p.Value
			case PointTypeActive:
				newCond.Active = FloatToBool(p.Value)
			}
		}
		ret.Conditions = append(ret.Conditions, newCond)
	}

	for _, act := range actionNodes {
		var newAct Action
		newAct.ID = act.ID
		for _, p := range act.Points {
			switch p.Type {
			case PointTypeDescription:
				newAct.Description = p.Text
			case PointTypeActionType:
				newAct.Action = p.Text
			case PointTypeID:
				newAct.NodeID = p.Text
			case PointTypePointType:
				newAct.PointType = p.Text
			case PointTypeValueType:
				newAct.PointValueType = p.Text
			case PointTypeValue:
				fmt.Println("COLLIN, value:", p.Value)
				newAct.PointValue = p.Value
				newAct.PointTextValue = p.Text
			case PointTypeFilePath:
				fmt.Println("COLLIN, file path:", p.Text)
				newAct.PointFilePath = p.Text
			}
		}
		ret.Actions = append(ret.Actions, newAct)
	}

	return ret, nil
}
