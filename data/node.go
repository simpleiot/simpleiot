package data

import (
	"time"
)

// don't even think about changing the below as it used
// in communications -- add new numbers
// if something needs changed/added.
const (
	SysStateUnknown  int = 0
	SysStatePowerOff     = 1
	SysStateOffline      = 2
	SysStateOnline       = 3
)

// SwUpdateState represents the state of an update
type SwUpdateState struct {
	Running     bool   `json:"running"`
	Error       string `json:"error"`
	PercentDone int    `json:"percentDone"`
}

// Node represents the state of a device. UUID is recommended
// for ID to prevent collisions is distributed instances.
type Node struct {
	ID     string   `json:"id" boltholdKey:"ID"`
	Type   string   `json:"type"`
	Points Points   `json:"points"`
	Groups []string `json:"groups"`
	Rules  []string `json:"rules"`
}

// Desc returns Description if set, otherwise ID
func (n *Node) Desc() string {
	desc, ok := n.Points.Text("", PointTypeDescription, 0)
	if ok && desc != "" {
		return desc
	}

	return n.ID
}

// SetState sets the device state
func (n *Node) SetState(state int) {
	n.ProcessPoint(Point{
		Time:  time.Now(),
		Type:  PointTypeSysState,
		Value: float64(state),
	})
}

// SetCmdPending for device
func (n *Node) SetCmdPending(pending bool) {
	val := 0.0
	if pending {
		val = 1
	}
	n.ProcessPoint(Point{
		Type:  PointTypeCmdPending,
		Value: val,
	})
}

// SetSwUpdateState for a device
func (n *Node) SetSwUpdateState(state SwUpdateState) {
	running := 0.0
	if state.Running {
		running = 1
	}
	n.ProcessPoint(Point{
		Type:  PointTypeSwUpdateRunning,
		Value: running,
	})

	n.ProcessPoint(Point{
		Type: PointTypeSwUpdateError,
		Text: state.Error,
	})

	n.ProcessPoint(Point{
		Type:  PointTypeSwUpdatePercComplete,
		Value: float64(state.PercentDone),
	})
}

// ProcessPoint takes a point for a device and adds/updates its array of points
func (n *Node) ProcessPoint(pIn Point) {
	pFound := false
	for i, p := range n.Points {
		if p.ID == pIn.ID && p.Type == pIn.Type && p.Index == pIn.Index {
			pFound = true
			n.Points[i] = pIn
		}
	}

	if !pFound {
		n.Points = append(n.Points, pIn)
	}
}

// UpdateState does routine updates of state (offline status, etc).
// Returns true if state was updated. We originally considered
// offline to be when we did not receive data from a remote device
// for X minutes. However, with points that could represent a config
// change as well. Eventually we may want to improve this to look
// at point types, but this is probably OK for now.
func (n *Node) UpdateState() (int, bool) {
	sysStateF, _ := n.Points.Value("", PointTypeSysState, 0)
	sysState := int(sysStateF)
	switch sysState {
	case SysStateUnknown, SysStateOnline:
		if time.Since(n.Points.LatestTime()) > 15*time.Minute {
			// mark device as offline
			n.SetState(SysStateOffline)
			return SysStateOffline, true
		}
	}

	return sysState, false
}

// State returns the current state of a device
func (n *Node) State() int {
	s, _ := n.Points.Value("", PointTypeSysState, 0)
	return int(s)
}

// ToUser converts a node to user struct
func (n *Node) ToUser() User {
	first, _ := n.Points.Text("", PointTypeFirstName, 0)
	last, _ := n.Points.Text("", PointTypeLastName, 0)
	phone, _ := n.Points.Text("", PointTypePhone, 0)
	email, _ := n.Points.Text("", PointTypeEmail, 0)
	pass, _ := n.Points.Text("", PointTypePass, 0)

	return User{
		ID:        n.ID,
		FirstName: first,
		LastName:  last,
		Phone:     phone,
		Email:     email,
		Pass:      pass,
	}
}

// define valid commands
const (
	CmdUpdateApp string = "updateApp"
	CmdPoll             = "poll"
	CmdFieldMode        = "fieldMode"
)

// NodeCmd represents a command to be sent to a device
type NodeCmd struct {
	ID     string `json:"id,omitempty" boltholdKey:"ID"`
	Cmd    string `json:"cmd"`
	Detail string `json:"detail,omitempty"`
}

// NodeVersion represents the device SW version
type NodeVersion struct {
	OS  string `json:"os"`
	App string `json:"app"`
	HW  string `json:"hw"`
}
