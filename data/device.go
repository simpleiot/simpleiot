package data

// DeviceConfig represents a device configuration (stuff that
// is set by user in UI)
type DeviceConfig struct {
	Description string `json:"description"`
}

// DeviceState represents information about a device that is
// collected, vs set by user.
type DeviceState struct {
	Version DeviceVersion `json:"version"`
	Ios     []Sample      `json:"ios"`
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
}

// ProcessSample takes a sample for a device and adds/updates in Ios
func (d *Device) ProcessSample(sample Sample) {
	ioFound := false
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
