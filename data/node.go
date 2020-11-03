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
// for ID. Parents is a list of devices this device is a child of. If
// Parents has a length of zero, this indicates it is a top level device.
// Groups and Rules likewise list groups and rules this device
// belongs to.
type Node struct {
	ID      string   `json:"id" boltholdKey:"ID"`
	Points  Points   `json:"points"`
	Parents []string `json:"devices"`
	Groups  []string `json:"groups"`
	Rules   []string `json:"rules"`
}

// Desc returns Description if set, otherwise ID
func (d *Node) Desc() string {
	desc, ok := d.Points.Text("", PointTypeDescription, 0)
	if ok && desc != "" {
		return desc
	}

	return d.ID
}

// SetState sets the device state
func (d *Node) SetState(state int) {
	d.ProcessPoint(Point{
		Type:  PointTypeSysState,
		Value: float64(state),
	})
}

// SetCmdPending for device
func (d *Node) SetCmdPending(pending bool) {
	val := 0.0
	if pending {
		val = 1
	}
	d.ProcessPoint(Point{
		Type:  PointTypeCmdPending,
		Value: val,
	})
}

// SetSwUpdateState for a device
func (d *Node) SetSwUpdateState(state SwUpdateState) {
	running := 0.0
	if state.Running {
		running = 1
	}
	d.ProcessPoint(Point{
		Type:  PointTypeSwUpdateRunning,
		Value: running,
	})

	d.ProcessPoint(Point{
		Type: PointTypeSwUpdateError,
		Text: state.Error,
	})

	d.ProcessPoint(Point{
		Type:  PointTypeSwUpdatePercComplete,
		Value: float64(state.PercentDone),
	})
}

// ProcessPoint takes a point for a device and adds/updates its array of points
func (d *Node) ProcessPoint(pIn Point) {
	pFound := false
	for i, p := range d.Points {
		if p.ID == pIn.ID && p.Type == pIn.Type && p.Index == pIn.Index {
			pFound = true
			d.Points[i] = pIn
		}
	}

	if !pFound {
		d.Points = append(d.Points, pIn)
	}
}

// UpdateState does routine updates of state (offline status, etc).
// Returns true if state was updated. We originally considered
// offline to be when we did not receive data from a remote device
// for X minutes. However, with points that could represent a config
// change as well. Eventually we may want to improve this to look
// at point types, but this is probably OK for now.
func (d *Node) UpdateState() (int, bool) {
	sysStateF, _ := d.Points.Value("", PointTypeSysState, 0)
	sysState := int(sysStateF)
	switch sysState {
	case SysStateUnknown, SysStateOnline:
		if time.Since(d.Points.LatestTime()) > 15*time.Minute {
			// mark device as offline
			d.SetState(SysStateOffline)
			return SysStateOffline, true
		}
	}

	return sysState, false
}

// State returns the current state of a device
func (d *Node) State() int {
	s, _ := d.Points.Value("", PointTypeSysState, 0)
	return int(s)
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
