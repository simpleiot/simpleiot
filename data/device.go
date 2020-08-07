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

// Device represents the state of a device
// The config is typically updated by the portal/UI, and the
// State is updated by the device. Keeping these datastructures
// separate reduces the possibility that one update will step
// on another.
type Device struct {
	ID         string       `json:"id" boltholdKey:"ID"`
	Config     DeviceConfig `json:"config"`
	State      DeviceState  `json:"state"`
	CmdPending bool         `json:"cmdPending"`
	Groups     []uuid.UUID  `json:"groups"`
	Rules      []uuid.UUID  `json:"rules"`
}

// Desc returns Description if set, otherwise ID
func (d *Device) Desc() string {
	if d.Config.Description != "" {
		return d.Config.Description
	}

	return d.ID
}

// ProcessSample takes a sample for a device and adds/updates in Ios
func (d *Device) ProcessSample(sample Sample) {
	ioFound := false
	// decide if sample is for IOs or device state
	if sample.ForDevice() {
		switch sample.Type {
		case SampleTypeSysState:
			d.State.SysState = SysState(sample.Value)
		}
	} else {
		for i, io := range d.State.Ios {
			if io.ID == sample.ID && io.Type == sample.Type {
				ioFound = true
				d.State.Ios[i] = sample
			}
		}

		if !ioFound {
			d.State.Ios = append(d.State.Ios, sample)
		}
	}
}

// UpdateState does routine updates of state (offline status, etc).
// Returns true if state was updated.
func (d *Device) UpdateState() bool {
	switch d.State.SysState {
	case SysStateUnknown, SysStateOnline:
		if time.Since(d.State.LastComm) > 15*time.Minute {
			// mark device as offline
			d.State.SysState = SysStateOffline
			return true

		}
	}

	return false
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
