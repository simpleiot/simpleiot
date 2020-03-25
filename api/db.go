package api

import (
	"fmt"
	"path"

	"github.com/simpleiot/simpleiot/data"
	"github.com/timshannon/bolthold"
	"github.com/google/uuid"
	"go.etcd.io/bbolt"
)

// Db is used for all db access in the application.
// We will eventually turn this into an interface to
// handle multiple Db backends.
type Db struct {
	store *bolthold.Store
}

// NewDb creates a new Db instance for the app
func NewDb(dataDir string) (*Db, error) {
	dbFile := path.Join(dataDir, "data.db")
	store, err := bolthold.Open(dbFile, 0666, nil)
	if err != nil {
		return nil, err
	}

	db := &Db{store: store}
	return db, db.init()
}

// DeviceUpdate updates a devices state in the database
func (db *Db) DeviceUpdate(device data.Device) error {
	return db.store.Upsert(device.ID, &device)
}

// deviceUpdateConfig updates the config for a particular device
func deviceUpdateConfig(store *bolthold.Store, id string, config data.DeviceConfig) error {
	return store.Bolt().Update(func(tx *bolt.Tx) error {
		var dev data.Device
		if err := store.TxGet(tx, id, &dev); err != nil {
			return err
		}

		dev.Config = config

		return store.TxUpdate(tx, id, dev)
	})
}

// DeviceSample processes a sample for a particular device
func (db *Db) DeviceSample(id string, sample data.Sample) error {
	return deviceSample(db.store, id, sample)
}

// deviceSample processes a sample for a particular device
func deviceSample(store *bolthold.Store, id string, sample data.Sample) error {
	return store.Bolt().Update(func(tx *bolt.Tx) error {
		var dev data.Device
		err := store.TxGet(tx, id, &dev)
		switch err {
		case bolthold.ErrNotFound:
			dev := data.Device{
				ID: id,
				State: data.DeviceState{
					Ios: []data.Sample{sample},
				},
			}

			return store.TxInsert(tx, id, dev)

		case nil:
			dev.ProcessSample(sample)
			return store.TxUpdate(tx, id, dev)
		}
		return err

	})
}

// Device returns data for a particular device
func (db *Db) Device(id string) (ret data.Device, err error) {
	err = db.store.Get(id, &ret)
	return
}

// DeviceDelete deletes a device from the database
func (db *Db) DeviceDelete(id string) error {
	return db.store.Delete(id, data.Device{})
}

// DeviceSetVersion sets a cmd for a device, and sets the
// CmdPending flag in the device structure.
func (db *Db) DeviceSetVersion(id string, ver data.DeviceVersion) error {

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
func (db *Db) Users() (ret []User, err error) {
	err = db.store.Find(&ret, nil)
	return
}

// User returns the user with the given ID, if it exists.
func (db *Db) User(id string) (user *data.User, err error) {
	var result []data.User
	err = db.store.Find(&result, bolthold.Where("ID").Eq(id))
	if err != nil {
		return nil, err
	}
	if len(result) != 1 {
		return nil, fmt.Errorf("no such user")
	}
	return &result[0], err
}

// UserUpsert modifies or creates a user.
func (db *Db) UserUpsert(id string, user data.User) error {
	return db.store.Upsert(user.ID, &user)
}

// Insert inserts a user with a new UUID.
func (db *Db) InsertUser(u User) (uuid.UUID, error) {
	id := uuid.New()
	return id, db.store.Insert(id, u)
}

func (db *Db) InsertOrg(o Org) (uuid.UUID, error) {
	id := uuid.New()
	return id, db.store.Insert(id, o)
}

func (db *Db) InsertRole(r Role) (uuid.UUID, error) {
	id := uuid.New()
	return id, db.store.Insert(id, r)
}

func (db *Db) Tx(fn func(*bolt.Tx)error) error {
	return db.store.Bolt().Update(fn)
}


// User provides information about a user.
type User struct {
	ID        uuid.UUID
	FirstName string
	LastName  string
	Email     string
	Pass      string
}

// A Role is the role played by a user in an organization.
type Role struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	OrgID       uuid.UUID
	Description string
}

// An Org is an organization.
type Org struct {
	ID   uuid.UUID
	Name string
}

// data.Device can be used in storage as well
// as on the wire, because its data is not relational.

// A Property is a relationship between a data.Device and an Org.
type Property struct {
	ID       uuid.UUID
	DeviceID uuid.UUID
	OrgID    uuid.UUID
}

// init initializes the database with one user (admin)
// in one organization (root).
// All devices are properties of the root organization.
func (db *Db) init() error {
	if ok, err := db.isInitialized(); err != nil {
		return err
	} else if ok {
		return nil
	}

	orgID, err := db.InsertOrg(Org{Name: "root"})
	if err != nil {
		return err
	}

	admin := User{
		FirstName: "admin",
		LastName:  "user",
		Email:     "admin@admin.com",
		Pass:      "admin",
	}

	userID, err := db.InsertUser(admin)
	if err != nil {
		return err
	}

	_, err = db.InsertRole(Role{
		UserID:      userID,
		OrgID:       orgID,
		Description: "admin",
	})
	return err
}

func (db *Db) isInitialized() (bool, error) {
	var orgs []Org
	if err := db.store.Find(&orgs, bolthold.Where("Name").Eq("root")); err != nil {
		return false, err
	}

	var roles []Role
	if err := db.store.Find(&roles, bolthold.Where("Description").Eq("admin")); err != nil {
		return false, err
	}

	return true, nil
}

//
// transformations between wire and storage types

func createUser(ds *Db, user data.User) (string, error) {
	// begin transaction
	id, err := ds.InsertUser(User{
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Pass:      user.Pass,
	})
	if err != nil {
		return "", err
	}

	for _, role := range user.Roles {
		if _, err := ds.InsertRole(Role{
			UserID:      id,
			OrgID:       role.OrgID,
			Description: role.Description,
		}); err != nil {
			return "", err
		}
	}
	// end transaction

	return id.String(), nil
}

