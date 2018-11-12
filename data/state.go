package data

import (
	"errors"
	"sync"
)

// DeviceSummary is a just returns device ID and description
type DeviceSummary struct {
	ID          string
	Description string
}

// DeviceState represents the state of a device
type DeviceState struct {
	ID          string
	Description string
	Ios         []Sample
}

// Summary returns summary information for a device state
func (ds DeviceState) Summary() DeviceSummary {
	return DeviceSummary{
		ID:          ds.ID,
		Description: ds.Description,
	}
}

// State represents the overall application state
type State struct {
	lock    sync.RWMutex
	devices []DeviceState
}

// Devices returns summary information for all devices
func (s *State) Devices() (ret []DeviceState) {
	for _, d := range s.devices {
		ret = append(ret, d)
	}
	return
}

// Device returns information for one device
func (s *State) Device(id string) (DeviceState, error) {
	for _, d := range s.devices {
		if d.ID == id {
			return d, nil
		}
	}

	return DeviceState{}, errors.New("device not found")
}

// UpdateDevice updates the state of a device with a sample
func (s *State) UpdateDevice(id string, sample Sample) {
	s.lock.Lock()
	defer s.lock.Unlock()

	deviceFound := false
	for i, d := range s.devices {
		if d.ID == id {
			deviceFound = true
			ioFound := false
			for j, io := range d.Ios {
				if io.ID == sample.ID {
					ioFound = true
					s.devices[i].Ios[j] = sample
				}
			}

			if !ioFound {
				s.devices[i].Ios = append(s.devices[i].Ios, sample)
			}
		}
	}
	if !deviceFound {
		s.devices = append(s.devices, DeviceState{
			ID: id,
			Ios: []Sample{
				sample,
			},
		})
	}
}
