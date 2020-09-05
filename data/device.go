package data

import (
	"time"

	"github.com/google/uuid"
)

// DeviceConfig represents a device configuration (stuff that
// is set by user in UI)
type DeviceConfig struct {
	Description string `json:"description"`
}

// SysState defines the system state
type SysState int

// define valid system states
// don't even think about changing the below as it used
// in communications -- add new numbers
// if something needs changed/added.
const (
	SysStateUnknown  SysState = 0
	SysStatePowerOff          = 1
	SysStateOffline           = 2
	SysStateOnline            = 3
)

// DeviceState represents information about a device that is
// collected, vs set by user.
type DeviceState struct {
	Version  DeviceVersion `json:"version"`
	Ios      []Sample      `json:"ios"`
	LastComm time.Time     `json:"lastComm"`
	SysState SysState      `json:"sysState"`
}

// SwUpdateState represents the state of an update
type SwUpdateState struct {
	Running     bool   `json:"running"`
	Error       string `json:"error"`
	PercentDone int    `json:"percentDone"`
}

// Device represents the state of a device. UUID is recommended
// for ID. Parents is a list of devices this device is a child of. If
// Parents has a length of zero, this indicates it is a top level device.
// Groups and Rules likewise list groups and rules this device
// belongs to.
type Device struct {
	ID      string      `json:"id" boltholdKey:"ID"`
	Points  Points      `json:"points"`
	Parents []string    `json:"devices"`
	Groups  []uuid.UUID `json:"groups"`
	Rules   []uuid.UUID `json:"rules"`
}

// Desc returns Description if set, otherwise ID
func (d *Device) Desc() string {
	desc, ok := d.Points.Text("", PointTypeDescription, 0)
	if ok && desc != "" {
		return desc
	}

	return d.ID
}

// SetState sets the device state
func (d *Device) SetState(state SysState) {
	d.ProcessPoint(Point{
		Type:  PointTypeSysState,
		Value: float64(state),
	})
}

// SetCmdPending for device
func (d *Device) SetCmdPending(pending bool) {
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
func (d *Device) SetSwUpdateState(state SwUpdateState) {
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
func (d *Device) ProcessPoint(pIn Point) {
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
func (d *Device) UpdateState() (SysState, bool) {
	sysStateF, _ := d.Points.Value("", PointTypeSysState, 0)
	sysState := SysState(sysStateF)
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
func (d *Device) State() SysState {
	s, _ := d.Points.Value("", PointTypeSysState, 0)
	return SysState(s)
}

// define valid commands
const (
	CmdUpdateApp string = "updateApp"
	CmdPoll             = "poll"
	CmdFieldMode        = "fieldMode"
)

// DeviceCmd represents a command to be sent to a device
type DeviceCmd struct {
	ID     string `json:"id,omitempty" boltholdKey:"ID"`
	Cmd    string `json:"cmd"`
	Detail string `json:"detail,omitempty"`
}

// DeviceVersion represents the device SW version
type DeviceVersion struct {
	OS  string `json:"os"`
	App string `json:"app"`
	HW  string `json:"hw"`
}
