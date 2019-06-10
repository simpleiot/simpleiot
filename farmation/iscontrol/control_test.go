package iscontrol

import (
	"testing"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

func TestExample(t *testing.T) {
	state := isdata.State{
		FlowRate: 20,
	}

	if !IsFlowGreater23(state) {
		t.Error("IsFlowGreater23 test failed")
	}
}
