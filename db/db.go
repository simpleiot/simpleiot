package db

import (
	"github.com/simpleiot/simpleiot/data"
	"github.com/timshannon/bolthold"
)

// Db is used for all db access in the application.
// We will eventually turn this into an interface to
// handle multiple Db backends.
type Db struct {
	store *bolthold.Store
}

// NewDb creates a new Db instance for the app
func NewDb() (*Db, error) {
	store, err := bolthold.Open("data.db", 0666, nil)
	if err != nil {
		return nil, err
	}

	return &Db{
		store: store,
	}, nil
}

// DeviceUpdate updates a devices state in the database
func (db *Db) DeviceUpdate(device data.Device) error {
	return db.store.Upsert(device.ID, &device)
}

// DeviceUpdateConfig updates the config for a particular device
func (db *Db) DeviceUpdateConfig(id string, config data.DeviceConfig) error {
	var dev data.Device
	err := db.store.Get(id, &dev)

	if err == bolthold.ErrNotFound {
		return err
	}

	dev.Config = config

	return db.store.Update(id, dev)
}

// DeviceSample processes a sample for a particular device
func (db *Db) DeviceSample(id string, sample data.Sample) error {
	var dev data.Device
	err := db.store.Get(id, &dev)

	if err == bolthold.ErrNotFound {
		dev := data.Device{
			ID: id,
			State: data.DeviceState{
				Ios: []data.Sample{sample},
			},
		}

		return db.store.Insert(id, dev)
	} else if err != nil {
		return err
	}

	dev.ProcessSample(sample)

	return db.store.Update(id, dev)
}

// Device returns data for a particular device
func (db *Db) Device(id string) (ret data.Device, err error) {
	err = db.store.Get(id, &ret)
	return
}

// Devices returns all devices
func (db *Db) Devices() (ret []data.Device, err error) {
	err = db.store.Find(&ret, nil)
	return
}
