package genji

import (
	"time"

	"github.com/simpleiot/simpleiot/data"
)

// This module is only used to import database
// dumps from old versions of software that
// still used the device data structure
// there is one manual fixup required in data.json
// file for rules -- s/deviceID/nodeid/gc

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
	Ios      []data.Point  `json:"ios"`
	LastComm time.Time     `json:"lastComm"`
	SysState SysState      `json:"sysState"`
}

// SwUpdateState represents the state of an update
type SwUpdateState struct {
	Running     bool   `json:"running"`
	Error       string `json:"error"`
	PercentDone int    `json:"percentDone"`
}

// Device represents the state of a device
// The config is typically updated by the portal/UI, and the
// State is updated by the device. Keeping these datastructures
// separate reduces the possibility that one update will step
// on another.
type Device struct {
	ID            string        `json:"id" boltholdKey:"ID"`
	Config        DeviceConfig  `json:"config"`
	State         DeviceState   `json:"state"`
	CmdPending    bool          `json:"cmdPending"`
	SwUpdateState SwUpdateState `json:"swUpdateState"`
	Groups        []string      `json:"groups"`
	Rules         []string      `json:"rules"`
}

// ToNode converts an old device type to current node type
func (d *Device) ToNode() data.Node {
	var node data.Node

	node.ID = d.ID

	node.Points = append(node.Points, d.State.Ios...)

	node.Points = append(node.Points,
		data.Point{
			Type: data.PointTypeDescription,
			Text: d.Config.Description,
		},
	)

	return node
}

// Desc returns Description if set, otherwise ID
func (d *Device) Desc() string {
	if d.Config.Description != "" {
		return d.Config.Description
	}

	return d.ID
}

// DeviceVersion represents the device SW version
type DeviceVersion struct {
	OS  string `json:"os"`
	App string `json:"app"`
	HW  string `json:"hw"`
}
