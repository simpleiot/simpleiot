package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"path"
	"time"

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
func NewDb(dataDir string, init bool) (*Db, error) {
	dbFile := path.Join(dataDir, "data.db")
	store, err := bolthold.Open(dbFile, 0666, nil)
	if err != nil {
		log.Println("bolthold open failed: ", err)
		return nil, err
	}

	db := &Db{store: store}
	if init {
		return db, initialize(db.store)
	}

	return db, nil
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

// deviceUpdateGroups updates the groups for a device.
func deviceUpdateGroups(store *bolthold.Store, id string, groups []uuid.UUID) error {
	return update(store, func(tx *bolt.Tx) error {
		var dev data.Device
		if err := store.TxGet(tx, id, &dev); err != nil {
			return err
		}

		dev.Groups = groups

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
		if err != nil {
			if err == bolthold.ErrNotFound {
				dev.ID = id
			} else {
				return err
			}
		}

		dev.ProcessSample(sample)
		return store.TxUpsert(tx, id, dev)
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

func deviceActivity(store *bolthold.Store, id string) error {
	return update(store, func(tx *bolt.Tx) error {
		var dev data.Device
		err := store.TxGet(tx, id, &dev)
		if err != nil {
			if err == bolthold.ErrNotFound {
				dev.ID = id
			} else {
				return err
			}
		}

		dev.State.LastComm = time.Now()
		dev.State.SysState = data.SysStateOnline
		return store.TxUpsert(tx, id, dev)
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

func deviceCmds(store *bolthold.Store) (ret []data.DeviceCmd, err error) {
	err = store.Find(&ret, nil)
	return
}

func devicesForUser(store *bolthold.Store, userID uuid.UUID) ([]data.Device, error) {
	var devices []data.Device

	isRoot, err := checkUserIsRoot(store, userID)
	if err != nil {
		return devices, err
	}

	if isRoot {
		// return all devices for root users
		err := store.Find(&devices, nil)
		return devices, err
	}

	err = view(store, func(tx *bolt.Tx) error {
		// First find groups users is part of
		var allGroups []data.Group
		err := store.TxFind(tx, &allGroups, nil)

		if err != nil {
			return err
		}

		var groupIDs []uuid.UUID

		for _, o := range allGroups {
			for _, ur := range o.Users {
				if ur.UserID == userID {
					groupIDs = append(groupIDs, o.ID)
				}
			}
		}

		// next, find devices that are part of the groups
		err = store.TxFind(tx, &devices,
			bolthold.Where("Groups").ContainsAny(bolthold.Slice(groupIDs)...))

		return nil
	})

	return devices, err
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

type privilege string

// check if user/password
func checkUser(store *bolthold.Store, email, password string) (*data.User, error) {
	var u data.User
	query := bolthold.Where("Email").Eq(email).
		And("Pass").Eq(password)
	err := store.FindOne(&u, query)
	if err != nil {
		if err != bolthold.ErrNotFound {
			return nil, err
		}
		return nil, nil
	}

	return &u, nil
}

// check is uses is port of the root group
func checkUserIsRoot(store *bolthold.Store, id uuid.UUID) (bool, error) {
	var group data.Group

	err := store.FindOne(&group, bolthold.Where("ID").Eq(zero))

	if err != nil {
		return false, err
	}

	for _, ur := range group.Users {
		if ur.UserID == id {
			return true, nil
		}
	}

	return false, nil

}

// userByID returns the user with the given ID, if it exists.
func userByID(store *bolthold.Store, id string) (data.User, error) {
	var ret data.User
	if err := store.FindOne(&ret, bolthold.Where("ID").Eq(id)); err != nil {
		return ret, err
	}

	return ret, nil
}

// userByEmail returns the user with the given email, if it exists.
func userByEmail(store *bolthold.Store, email string) (data.User, error) {
	var ret data.User
	if err := store.FindOne(&ret, bolthold.Where("Email").Eq(email)); err != nil {
		return ret, err
	}

	return ret, nil
}

// initialize initializes the database with one user (admin)
// in one groupanization (root).
// All devices are properties of the root groupanization.
func initialize(store *bolthold.Store) error {
	// initialize root group in new db
	var group data.Group
	err := store.FindOne(&group, bolthold.Where("Name").Eq("root"))

	// group was found or we ran into an error
	if err != bolthold.ErrNotFound {
		return err
	}

	// add root group and admin user
	return update(store, func(tx *bolt.Tx) error {
		log.Println("adding root group and admin user ...")

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

		group := data.Group{
			ID:   zero,
			Name: "root",
			Users: []data.UserRoles{
				{UserID: zero, Roles: []data.Role{data.RoleAdmin}},
			},
		}

		if err := store.TxInsert(tx, group.ID, group); err != nil {
			return err
		}

		log.Println("Created root group:", group)
		return nil
	})
}

// groupDevices returns the devices which are property of the given Group.
func groupDevices(store *bolthold.Store, tx *bolt.Tx, groupID uuid.UUID) ([]data.Device, error) {
	var devices []data.Device
	err := view(store, func(tx *bolt.Tx) error {
		if err := store.TxFind(tx, &devices, bolthold.Where("Groups").Contains(groupID)); err != nil {
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
			log.Printf("Error updating user %v, try fixing key\n", user.Email)

			// Delete current user with bad key
			err := store.TxDeleteMatching(tx, data.User{},
				bolthold.Where("ID").Eq(user.ID))

			if err != nil {
				log.Println("Error deleting user when trying to fix up: ", err)
				return err
			}

			// try to insert group
			if err = store.TxUpsert(tx, user.ID, user); err != nil {
				log.Println("Error updating user after delete: ", err)
				return err
			}

			return err
		}

		return nil
	})
}

// deleteUser deletes a user from the database
func deleteUser(store *bolthold.Store, id uuid.UUID) error {
	return store.Delete(id, data.User{})
}

// Groups returns all groups.
func groups(store *bolthold.Store) ([]data.Group, error) {
	var ret []data.Group
	if err := store.Find(&ret, nil); err != nil {
		return ret, fmt.Errorf("problem finding groups: %v", err)
	}

	return ret, nil
}

// group returns the Group with the given ID.
func group(store *bolthold.Store, id uuid.UUID) (data.Group, error) {
	var group data.Group
	err := store.FindOne(&group, bolthold.Where("ID").Eq(id))
	return group, err
}

func insertGroup(store *bolthold.Store, group data.Group) (string, error) {
	id := uuid.New()

	group.Parent = zero

	err := update(store, func(tx *bolt.Tx) error {
		if err := store.TxInsert(tx, id, group); err != nil {
			return err
		}

		return nil
	})

	return id.String(), err
}

func updateGroup(store *bolthold.Store, gUpdate data.Group) error {
	return update(store, func(tx *bolt.Tx) error {
		if err := store.TxUpdate(tx, gUpdate.ID, gUpdate); err != nil {
			log.Printf("Error updating group %v, try fixing key\n", gUpdate.Name)

			// Delete current group with bad key
			err := store.TxDeleteMatching(tx, data.Group{},
				bolthold.Where("ID").Eq(gUpdate.ID))

			if err != nil {
				log.Println("Error deleting group when trying to fix up: ", err)
				return err
			}

			// try to insert group
			if err = store.TxUpsert(tx, gUpdate.ID, gUpdate); err != nil {
				log.Println("Error updating group after delete: ", err)
				return err
			}
		}

		return nil
	})
}

// deleteGroup deletes a device from the database
func deleteGroup(store *bolthold.Store, id uuid.UUID) error {
	return store.Delete(id, data.Group{})
}

func newIfZero(id uuid.UUID) uuid.UUID {
	if id == zero {
		return uuid.New()
	}
	return id
}

type dbDump struct {
	Devices    []data.Device    `json:"devices"`
	Users      []data.User      `json:"users"`
	Groups     []data.Group     `json:"groups"`
	DeviceCmds []data.DeviceCmd `json:"deviceCmds"`
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

	dump.Groups, err = groups(db.store)
	if err != nil {
		return err
	}

	dump.DeviceCmds, err = deviceCmds(db.store)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "   ")

	return encoder.Encode(dump)
}
