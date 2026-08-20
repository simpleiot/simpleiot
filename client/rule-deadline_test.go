package client

import (
	"testing"
	"time"
)

// nextDeadline reports the earliest moment the rule can change state on its
// own. Nothing pending means no timer runs at all.
func TestRuleNextDeadline(t *testing.T) {
	now := time.Now()

	rc := &RuleClient{
		config: Rule{
			ID: "rule",
			Conditions: []Condition{
				{ID: "c1"},
				{ID: "c2"},
				{ID: "c3", Disabled: true},
			},
		},
		condState: map[string]*condRuntime{},
	}

	if got := rc.nextDeadline(); !got.IsZero() {
		t.Errorf("expected no deadline with no condition state, got %v", got)
	}

	rc.condState["c1"] = &condRuntime{}
	rc.condState["c2"] = &condRuntime{}

	if got := rc.nextDeadline(); !got.IsZero() {
		t.Errorf("expected no deadline when nothing is pending, got %v", got)
	}

	rc.condState["c1"].deadline = now.Add(30 * time.Second)

	if got := rc.nextDeadline(); !got.Equal(now.Add(30 * time.Second)) {
		t.Errorf("expected the only pending deadline, got %v", got)
	}

	rc.condState["c2"].deadline = now.Add(5 * time.Second)

	if got := rc.nextDeadline(); !got.Equal(now.Add(5 * time.Second)) {
		t.Errorf("expected the earliest pending deadline, got %v", got)
	}

	// a disabled condition cannot move the rule, so its deadline is ignored
	rc.condState["c3"] = &condRuntime{deadline: now.Add(time.Second)}

	if got := rc.nextDeadline(); !got.Equal(now.Add(5 * time.Second)) {
		t.Errorf("expected a disabled condition's deadline to be ignored, got %v", got)
	}
}
