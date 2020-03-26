package api

import (
	"fmt"
	"log"
	"path"

	"github.com/google/uuid"
	"github.com/simpleiot/simpleiot/data"
	"github.com/timshannon/bolthold"
	"go.etcd.io/bbolt"
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

// deviceSample processes a sample for a particular device
func deviceSample(store *bolthold.Store, id string, sample data.Sample) error {
	return update(store, func(tx *bolt.Tx) error {
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
		var users []User
		if err := store.TxFind(tx, &users, nil); err != nil {
			return err
		}

		ret = make([]data.User, len(users))
		for i, user := range users {
			r, err := userRolesData(store, tx, user.ID)
			if err != nil {
				return err
			}

			ret[i] = data.User{
				ID:        user.ID,
				FirstName: user.FirstName,
				LastName:  user.LastName,
				Email:     user.Email,
				Pass:      user.Pass,
				Roles:     r,
			}
		}
		return nil
	})
	return ret, err
}

func org(store *bolthold.Store, tx *bolt.Tx, id uuid.UUID) (Org, error) {
	var org Org
	err := store.TxFindOne(tx, &org, bolthold.Where("ID").Eq(id))
	return org, err
}

func userRolesData(store *bolthold.Store, tx *bolt.Tx, id uuid.UUID) ([]data.Role, error) {
	var r []data.Role
	roles, err := userRoles(store, tx, id)
	if err != nil {
		return r, err
	}

	r = make([]data.Role, len(roles))
	for i, role := range roles {
		org, err := org(store, tx, role.OrgID)
		if err != nil {
			return r, err
		}

		r[i] = data.Role{
			ID:          role.ID,
			OrgID:       role.OrgID,
			OrgName:     org.Name,
			Description: role.Description,
		}
	}
	return r, nil
}

func validLogin(store *bolthold.Store, email, password string) (bool, error) {
	var users []User
	query := bolthold.Where("Email").Eq(email).
		And("Pass").Eq(password)
	err := store.Find(&users, query)
	switch err {
	case bolthold.ErrNotFound:
		return false, nil

	case nil:
		return true, nil
	}

	return false, err
}

func userRoles(store *bolthold.Store, tx *bolt.Tx, id uuid.UUID) ([]Role, error) {
	var roles []Role
	err := store.TxFind(tx, &roles, bolthold.Where("UserID").Eq(id))
	return roles, err
}

// user returns the user with the given ID, if it exists.
func user(store *bolthold.Store, id string) (*data.User, error) {
	var ret *data.User
	err := view(store, func(tx *bolt.Tx) error {
		var user User
		if err := store.TxFindOne(tx, &user, bolthold.Where("ID").Eq(id)); err != nil {
			return err
		}

		roles, err := userRoles(store, tx, user.ID)
		if err != nil {
			return err
		}

		r := make([]data.Role, len(roles))
		for i, role := range roles {
			var org Org
			if err := store.TxFindOne(tx, &org, bolthold.Where("ID").Eq(role.OrgID)); err != nil {
				return err
			}
			r[i] = data.Role{
				ID:          role.ID,
				OrgID:       role.OrgID,
				OrgName:     org.Name,
				Description: role.Description,
			}
		}

		ret = &data.User{
			ID:        user.ID,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Email:     user.Email,
			Pass:      user.Pass,
			Roles:     r,
		}
		return nil
	})
	return ret, err
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

// initialize initializes the database with one user (admin)
// in one organization (root).
// All devices are properties of the root organization.
func initialize(store *bolthold.Store) error {
	if ok, err := isInitialized(store); err != nil {
		return err
	} else if ok {
		return nil
	}

	return update(store, func(tx *bolt.Tx) error {
		root := Org{
			ID:   uuid.New(),
			Name: "root",
		}
		if err := store.TxInsert(tx, root.ID, root); err != nil {
			return err
		}
		log.Println(root)

		admin := User{
			ID:        uuid.New(),
			FirstName: "admin",
			LastName:  "user",
			Email:     "admin@admin.com",
			Pass:      "admin",
		}

		if err := store.TxInsert(tx, admin.ID, admin); err != nil {
			return err
		}
		log.Println(admin)

		role := Role{
			ID:          uuid.New(),
			UserID:      admin.ID,
			OrgID:       root.ID,
			Description: "admin",
		}
		defer log.Println(role)
		return store.TxInsert(tx, role.ID, role)
	})
}

func isInitialized(store *bolthold.Store) (bool, error) {
	// Is there an organization called root?
	root := bolthold.Where("Name").Eq("root")
	var org Org
	switch err := store.FindOne(&org, root); err {
	case nil:
		// OK

	case bolthold.ErrNotFound:
		return false, nil

	default:
		return false, fmt.Errorf("error checking whether database is initialized: %v", err)
	}

	// Does the root organization have an admin?
	var role Role
	admin := bolthold.Where("Description").Eq("admin").And("OrgID").Eq(org.ID)
	if err := store.FindOne(&role, admin); err != nil {
		return false, fmt.Errorf("error checking whether database is initialized: found a root org, but not an admin: %v", err)
	}

	return true, nil
}

func insertUser(store *bolthold.Store, user data.User) (string, error) {
	u := User{
		ID:        uuid.New(),
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Pass:      user.Pass,
	}

	err := update(store, func(tx *bolt.Tx) error {
		if err := store.TxInsert(tx, u.ID, user); err != nil {
			return err
		}

		for _, role := range user.Roles {
			role := Role{
				ID:          uuid.New(),
				UserID:      u.ID,
				OrgID:       role.OrgID,
				Description: role.Description,
			}
			if err := store.TxInsert(tx, role.ID, role); err != nil {
				return err
			}
		}
		return nil
	})

	return u.ID.String(), err
}

func updateUser(store *bolthold.Store, user data.User) error {
	u := User{
		ID:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Pass:      user.Pass,
	}

	return update(store, func(tx *bolt.Tx) error {
		if err := store.TxUpdate(tx, u.ID, user); err != nil {
			return err
		}

		// TODO: What if the set of roles changes?
		// This code is incorrect! Consider embedding roles.
		for _, role := range user.Roles {
			role := Role{
				ID:          role.ID,
				UserID:      u.ID,
				OrgID:       role.OrgID,
				Description: role.Description,
			}
			if err := store.TxUpdate(tx, role.ID, role); err != nil {
				return err
			}
		}
		return nil
	})
}
