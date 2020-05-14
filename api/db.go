package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"path"

	"github.com/google/uuid"
	"github.com/simpleiot/simpleiot/data"
	"github.com/timshannon/bolthold"
	bolt "go.etcd.io/bbolt"
)

// This file contains database manipulations.

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
	return db, initialize(db.store)
}

// deviceUpdateConfig updates the config for a device.
func deviceUpdateConfig(store *bolthold.Store, id string, config data.DeviceConfig) error {
	return update(store, func(tx *bolt.Tx) error {
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

func update(store *bolthold.Store, fn func(tx *bolt.Tx) error) error {
	return store.Bolt().Update(fn)
}

func view(store *bolthold.Store, fn func(tx *bolt.Tx) error) error {
	return store.Bolt().View(fn)
}

var zero uuid.UUID

// deviceSample processes a sample for a particular device
func deviceSample(store *bolthold.Store, id string, sample data.Sample) error {
	return update(store, func(tx *bolt.Tx) error {
		var dev data.Device
		err := store.TxGet(tx, id, &dev)
		switch err {
		case bolthold.ErrNotFound:
			// New devices are automatically part of root org
			dev := data.Device{
				ID: id,
				State: data.DeviceState{
					Ios: []data.Sample{sample},
				},
				Orgs: []uuid.UUID{zero},
			}

			return store.TxInsert(tx, id, dev)

		case nil:
			dev.ProcessSample(sample)
			return store.TxUpdate(tx, id, dev)
		}
		return err
	})
}

// device returns data for a particular device
func device(store *bolthold.Store, id string) (ret data.Device, err error) {
	err = store.Get(id, &ret)
	return
}

// deviceDelete deletes a device from the database
func deviceDelete(store *bolthold.Store, id string) error {
	return store.Delete(id, data.Device{})
}

// deviceSetVersion sets a cmd for a device, and sets the
// CmdPending flag in the device structure.
func deviceSetVersion(store *bolthold.Store, id string, ver data.DeviceVersion) error {
	return update(store, func(tx *bolt.Tx) error {
		var dev data.Device
		err := store.TxGet(tx, id, &dev)
		if err != nil {
			return err
		}

		dev.State.Version = ver
		return store.TxUpdate(tx, id, dev)
	})
}

// deviceSetCmd sets a cmd for a device, and sets the
// CmdPending flag in the device structure.
func deviceSetCmd(store *bolthold.Store, cmd data.DeviceCmd) error {
	return update(store, func(tx *bolt.Tx) error {
		err := store.TxUpsert(tx, cmd.ID, &cmd)
		if err != nil {
			return err
		}

		// and clear the device pending flag
		var dev data.Device
		err = store.TxGet(tx, cmd.ID, &dev)
		if err != nil {
			return err
		}

		dev.CmdPending = true
		return store.TxUpdate(tx, cmd.ID, dev)
	})
}

// DeviceGetCmd gets a cmd for a device. If the cmd is no null,
// the command is deleted, and the cmdPending flag cleared in
// the Device data structure.
func deviceGetCmd(store *bolthold.Store, id string) (data.DeviceCmd, error) {
	var cmd data.DeviceCmd
	err := store.Get(id, &cmd)
	if err == bolthold.ErrNotFound {
		// we don't consider this an error in this case
		err = nil
	}

	if err != nil {
		return cmd, err
	}

	if cmd.Cmd != "" {
		// a device has fetched a command, delete it
		err := store.Delete(id, data.DeviceCmd{})
		if err != nil {
			return cmd, err
		}

		// and clear the device pending flag
		var dev data.Device
		err = store.Get(id, &dev)
		if err != nil {
			return cmd, err
		}

		dev.CmdPending = false
		err = store.Update(id, dev)
		if err != nil {
			return cmd, err
		}
	}

	return cmd, err
}

// devices returns all devices.
func devices(store *bolthold.Store) (ret []data.Device, err error) {
	err = store.Find(&ret, nil)
	return
}

// Users returns all users.
func users(store *bolthold.Store) ([]data.User, error) {
	var ret []data.User
	err := view(store, func(tx *bolt.Tx) error {
		if err := store.TxFind(tx, &ret, nil); err != nil {
			return err
		}
		return nil
	})
	return ret, err
}

// org returns the Org with the given ID.
func org(store *bolthold.Store, id uuid.UUID) (data.Org, error) {
	var org data.Org
	err := store.FindOne(&org, bolthold.Where("ID").Eq(id))
	return org, err
}

type privilege string

// check if user exists
func checkUser(store *bolthold.Store, email, password string) (bool, error) {
	var u data.User
	query := bolthold.Where("Email").Eq(email).
		And("Pass").Eq(password)
	err := store.FindOne(&u, query)
	if err != nil {
		if err != bolthold.ErrNotFound {
			return false, err
		}
		return false, nil
	}

	return true, nil
}

// userByID returns the user with the given ID, if it exists.
func userByID(store *bolthold.Store, id string) (*data.User, error) {
	var ret *data.User
	err := view(store, func(tx *bolt.Tx) error {
		var user data.User
		if err := store.TxFindOne(tx, &user, bolthold.Where("ID").Eq(id)); err != nil {
			return err
		}

		return nil
	})
	return ret, err
}

// initialize initializes the database with one user (admin)
// in one organization (root).
// All devices are properties of the root organization.
func initialize(store *bolthold.Store) error {
	// initialize root org in new db
	var org data.Org
	err := store.FindOne(&org, bolthold.Where("Name").Eq("root"))

	// org was found or we ran into an error
	if err != bolthold.ErrNotFound {
		return err
	}

	// add root org and admin user
	return update(store, func(tx *bolt.Tx) error {

		admin := data.User{
			ID:        zero,
			FirstName: "admin",
			LastName:  "user",
			Email:     "admin@admin.com",
			Pass:      "admin",
		}

		if err := store.TxInsert(tx, admin.ID, admin); err != nil {
			return err
		}

		log.Println("Created admin user: ", admin)

		org := data.Org{
			ID:   zero,
			Name: "root",
			Users: []data.UserRoles{
				{UserID: zero, Roles: []data.Role{data.RoleAdmin}},
			},
		}

		if err := store.TxInsert(tx, org.ID, org); err != nil {
			return err
		}

		log.Println("Created root org:", org)
		return nil
	})
}

// orgDevices returns the devices which are property of the given Org.
func orgDevices(store *bolthold.Store, tx *bolt.Tx, orgID uuid.UUID) ([]data.Device, error) {
	var devices []data.Device
	err := view(store, func(tx *bolt.Tx) error {
		if err := store.TxFind(tx, &devices, bolthold.Where("Orgs").Contains(orgID)); err != nil {
			return err
		}

		return nil
	})
	return devices, err
}

func insertUser(store *bolthold.Store, user data.User) (string, error) {
	id := uuid.New()

	err := update(store, func(tx *bolt.Tx) error {
		if err := store.TxInsert(tx, id, user); err != nil {
			return err
		}

		return nil
	})

	return id.String(), err
}

func updateUser(store *bolthold.Store, user data.User) error {
	return update(store, func(tx *bolt.Tx) error {
		if err := store.TxUpdate(tx, user.ID, user); err != nil {
			return err
		}

		return nil
	})
}

func insertOrg(store *bolthold.Store, org data.Org) (string, error) {
	id := uuid.New()

	org.Parent = zero

	err := update(store, func(tx *bolt.Tx) error {
		if err := store.TxInsert(tx, id, org); err != nil {
			return err
		}

		return nil
	})

	return id.String(), err
}

func updateOrg(store *bolthold.Store, org data.Org) error {
	return update(store, func(tx *bolt.Tx) error {
		if err := store.TxUpdate(tx, org.ID, org); err != nil {
			return err
		}

		return nil
	})
}

func newIfZero(id uuid.UUID) uuid.UUID {
	if id == zero {
		return uuid.New()
	}
	return id
}

// Orgs returns all orgs.
func orgs(store *bolthold.Store) ([]data.Org, error) {
	var ret []data.Org
	err := view(store, func(tx *bolt.Tx) error {
		if err := store.TxFind(tx, &ret, nil); err != nil {
			return fmt.Errorf("problem finding orgs: %v", err)
		}

		return nil
	})
	return ret, err
}

type dbDump struct {
	Devices []data.Device `json:"devices"`
	Users   []data.User   `json:"users"`
	Orgs    []data.Org    `json:"orgs"`
}

// DumpDb dumps the entire db to a file
func DumpDb(db *Db, out io.Writer) error {
	dump := dbDump{}

	var err error

	dump.Devices, err = devices(db.store)
	if err != nil {
		return err
	}

	dump.Users, err = users(db.store)
	if err != nil {
		return err
	}

	dump.Orgs, err = orgs(db.store)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "   ")

	return encoder.Encode(dump)
}
