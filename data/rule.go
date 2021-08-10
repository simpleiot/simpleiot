package data

import (
	"fmt"
	"time"
)

// Condition defines parameters to look for in a point or a schedule.
type Condition struct {
	// general parameters
	ID            string
	Description   string
	ConditionType string
	MinTimeActive float64
	Active        bool

	// used with point value rules
	NodeID         string
	PointType      string
	PointID        string
	PointIndex     int
	PointValueType string
	Operator       string
	PointValue     float64
	PointTextValue string

	// used with shedule rules
	StartTime string
	EndTime   string
	Weekdays  []time.Weekday
}

func (c Condition) String() string {
	ret := fmt.Sprintf("  COND: %v, V:%v, A:%v\n", c.Description, c.PointValue, c.Active)
	return ret
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
}

func (a Action) String() string {
	ret := fmt.Sprintf("  ACTION: %v, %v\n", a.Description, a.PointValue)
	return ret
}

// RuleConfig contains parts of the rule that a users changes
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

func (r Rule) String() string {
	ret := fmt.Sprintf("Rule: %v\n", r.Description)
	ret += fmt.Sprintf("  active: %v\n", r.Active)
	for _, c := range r.Conditions {
		ret += fmt.Sprintf("%v", c)
	}
	for _, a := range r.Actions {
		ret += fmt.Sprintf("%v", a)
	}

	return ret
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
			case PointTypeConditionType:
				newCond.ConditionType = p.Text
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
			case PointTypeStart:
				newCond.StartTime = p.Text
			case PointTypeEnd:
				newCond.EndTime = p.Text
			case PointTypeWeekday:
				if p.Value > 0 {
					newCond.Weekdays = append(newCond.Weekdays, time.Weekday(p.Index))
				}
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
				newAct.PointValue = p.Value
				newAct.PointTextValue = p.Text
			}
		}
		ret.Actions = append(ret.Actions, newAct)
	}

	return ret, nil
}
