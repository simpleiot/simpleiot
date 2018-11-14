package data

// Device represents the state of a device
type Device struct {
	ID          string   `json:"id" boltholdKey:"ID"`
	Description string   `json:"description"`
	Ios         []Sample `json:"ios"`
}

// ProcessSample takes a sample for a device and adds/updates in Ios
func (d *Device) ProcessSample(sample Sample) {
	ioFound := false
	for i, io := range d.Ios {
		if io.ID == sample.ID {
			ioFound = true
			d.Ios[i] = sample
		}
	}

	if !ioFound {
		d.Ios = append(d.Ios, sample)
	}
}
