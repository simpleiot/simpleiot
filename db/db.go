package db

import (
	"log"
	"path"
	"sync"

	"github.com/google/uuid"
	"github.com/simpleiot/simpleiot/data"
	"github.com/timshannon/bolthold"
)

// Db is used for all db access in the application.
// We will eventually turn this into an interface to
// handle multiple Db backends.
type Db struct {
	store *bolthold.Store
	lock  sync.RWMutex
}

// NewDb creates a new Db instance for the app
func NewDb(dataDir string) (*Db, error) {
	dbFile := path.Join(dataDir, "data.db")
	store, err := bolthold.Open(dbFile, 0666, nil)
	if err != nil {
		return nil, err
	}

	// make sure there is one user, otherwise add admin user
	var users []data.User
	err = store.Find(&users, nil)

	if len(users) <= 0 {
		log.Println("Creating admin user")
		err = store.Insert(
			bolthold.NextSequence(), data.User{
				ID:        uuid.New(),
				FirstName: "admin",
				LastName:  "user",
				Email:     "admin@admin.com",
				Admin:     true,
				Pass:      "admin",
			})

		if err != nil {
			return nil, err
		}
	}

	return &Db{
		store: store,
	}, nil
}

// DeviceUpdate updates a devices state in the database
func (db *Db) DeviceUpdate(device data.Device) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	return db.store.Upsert(device.ID, &device)
}

// DeviceUpdateConfig updates the config for a particular device
func (db *Db) DeviceUpdateConfig(id string, config data.DeviceConfig) error {
	db.lock.Lock()
	defer db.lock.Unlock()
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
	db.lock.Lock()
	defer db.lock.Unlock()
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
	db.lock.RLock()
	defer db.lock.RUnlock()
	err = db.store.Get(id, &ret)
	return
}

// DeviceDelete deletes a device from the database
func (db *Db) DeviceDelete(id string) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	return db.store.Delete(id, data.Device{})
}

// DeviceSetVersion sets a cmd for a device, and sets the
// CmdPending flag in the device structure.
func (db *Db) DeviceSetVersion(id string, ver data.DeviceVersion) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	var dev data.Device
	err := db.store.Get(id, &dev)
	if err != nil {
		return err
	}

	dev.State.Version = ver
	return db.store.Update(id, dev)
}

// DeviceSetCmd sets a cmd for a device, and sets the
// CmdPending flag in the device structure.
func (db *Db) DeviceSetCmd(cmd data.DeviceCmd) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	err := db.store.Upsert(cmd.ID, &cmd)
	if err != nil {
		return err
	}

	// and clear the device pending flag
	var dev data.Device
	err = db.store.Get(cmd.ID, &dev)
	if err != nil {
		return err
	}

	dev.CmdPending = true
	return db.store.Update(cmd.ID, dev)
}

// DeviceGetCmd gets a cmd for a device. If the cmd is no null,
// the command is deleted, and the cmdPending flag cleared in
// the Device data structure.
func (db *Db) DeviceGetCmd(id string) (data.DeviceCmd, error) {
	var cmd data.DeviceCmd
	err := db.store.Get(id, &cmd)
	if err == bolthold.ErrNotFound {
		// we don't consider this an error in this case
		err = nil
	}

	if err != nil {
		return cmd, err
	}

	if cmd.Cmd != "" {
		// a device has fetched a command, delete it
		db.lock.Lock()
		defer db.lock.Unlock()
		err := db.store.Delete(id, data.DeviceCmd{})
		if err != nil {
			return cmd, err
		}

		// and clear the device pending flag
		var dev data.Device
		err = db.store.Get(id, &dev)
		if err != nil {
			return cmd, err
		}

		dev.CmdPending = false
		err = db.store.Update(id, dev)
		if err != nil {
			return cmd, err
		}
	}

	return cmd, err
}

// Devices returns all devices
func (db *Db) Devices() (ret []data.Device, err error) {
	err = db.store.Find(&ret, nil)
	return
}

// Users returns all users.
func (db *Db) Users() (ret []data.User, err error) {
	err = db.store.Find(&ret, nil)
	return
}
