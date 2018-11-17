package data

// DeviceConfig represents a device configuration (stuff that
// is set by user in UI)
type DeviceConfig struct {
	Description string `json:"description"`
}

// DeviceState represents information about a device that is
// collected, vs set by user.
type DeviceState struct {
	Ios []Sample `json:"ios"`
}

// Device represents the state of a device
type Device struct {
	ID     string       `json:"id" boltholdKey:"ID"`
	Config DeviceConfig `json:"config"`
	State  DeviceState  `json:"state"`
}

// ProcessSample takes a sample for a device and adds/updates in Ios
func (d *Device) ProcessSample(sample Sample) {
	ioFound := false
	for i, io := range d.State.Ios {
		if io.ID == sample.ID {
			ioFound = true
			d.State.Ios[i] = sample
		}
	}

	if !ioFound {
		d.State.Ios = append(d.State.Ios, sample)
	}
}
