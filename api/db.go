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
			dev := data.Device{
				ID: id,
				State: data.DeviceState{
					Ios: []data.Sample{sample},
				},
			}

			if err := store.TxInsert(tx, id, dev); err != nil {
				return err
			}

			// New devices are properties of root.
			prop := Property{
				ID:       uuid.New(),
				DeviceID: id,
				OrgID:    zero,
			}

			return store.TxInsert(tx, prop.ID, prop)

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
func org(store *bolthold.Store, tx *bolt.Tx, id uuid.UUID) (Org, error) {
	var org Org
	err := store.TxFindOne(tx, &org, bolthold.Where("ID").Eq(id))
	return org, err
}

type privilege string

const (
	none  privilege = ""
	user            = "user"
	admin           = "admin"
	root            = "root"
)

// loginPrivilege returns the privilege level
// associated with the given email-password combination.
func loginPrivilege(store *bolthold.Store, email, password string) (privilege, error) {
	var u User
	query := bolthold.Where("Email").Eq(email).
		And("Pass").Eq(password)
	err := store.FindOne(&u, query)
	switch err {
	case bolthold.ErrNotFound:
		return none, nil

	case nil:
		switch u.ID {
		case zero:
			return root, nil
		}

		return user, nil
	}

	return none, err
}

// userRoles returns the roles played by the user with the given ID.
func userRoles(store *bolthold.Store, tx *bolt.Tx, id uuid.UUID) ([]Role, error) {
	var roles []Role
	err := store.TxFind(tx, &roles, bolthold.Where("UserID").Eq(id))
	return roles, err
}

// userByID returns the user with the given ID, if it exists.
func userByID(store *bolthold.Store, id string) (*data.User, error) {
	var ret *data.User
	err := view(store, func(tx *bolt.Tx) error {
		var user User
		if err := store.TxFindOne(tx, &user, bolthold.Where("ID").Eq(id)); err != nil {
			return err
		}

		return nil
	})
	return ret, err
}

// User provides information about a user.
type User struct {
	ID        uuid.UUID `boltholdKey:"ID"`
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
	DeviceID string
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
			ID:   zero,
			Name: "root",
		}
		if err := store.TxInsert(tx, root.ID, root); err != nil {
			return err
		}
		log.Println(root)

		admin := User{
			ID:        zero,
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
			ID:          zero,
			UserID:      admin.ID,
			OrgID:       root.ID,
			Description: "admin",
		}
		defer func() {
			log.Println(role)
			var u User
			if err := store.TxGet(tx, admin.ID, &u); err != nil {
				log.Println(err)
			} else {
				log.Println(u)
			}
		}()
		return store.TxInsert(tx, role.ID, role)
	})
}

func isInitialized(store *bolthold.Store) (ok bool, err error) {
	err = view(store, func(tx *bolt.Tx) error {
		// Is there an organization called root?
		root := bolthold.Where("Name").Eq("root")
		var org Org
		switch err := store.TxFindOne(tx, &org, root); err {
		case nil:
			// OK
			log.Printf("found root org: %v", org)

		case bolthold.ErrNotFound:
			return nil

		default:
			return fmt.Errorf("error checking whether database is initialized: %v", err)
		}

		// Does the root organization have an admin?
		var role Role
		admin := bolthold.Where("Description").Eq("admin").And("OrgID").Eq(org.ID)
		if err := store.TxFindOne(tx, &role, admin); err != nil {
			return fmt.Errorf("error checking whether database is initialized: found a root org, but not an admin: %v", err)
		}

		log.Printf("found admin role: %v", role)

		var user User
		if err := store.TxGet(tx, role.UserID, &user); err != nil {
			return fmt.Errorf("error checking whether database is initialized: found admin role, but no user: %v", err)
		}
		log.Printf("found admin user: %v", user)
		ok = true
		return nil
	})
	return
}

// orgDevices returns the devices which are property of the given Org.
func orgDevices(store *bolthold.Store, tx *bolt.Tx, orgID uuid.UUID) ([]data.Device, error) {
	var devices []data.Device
	err := view(store, func(tx *bolt.Tx) error {
		var props []Property
		if err := store.TxFind(tx, &props, bolthold.Where("OrgID").Eq(orgID)); err != nil {
			return err
		}

		for _, prop := range props {
			var device data.Device
			if err := store.TxGet(tx, prop.DeviceID, &device); err != nil {
				return err
			}
			devices = append(devices, device)
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

func newIfZero(id uuid.UUID) uuid.UUID {
	if id == zero {
		return uuid.New()
	}
	return id
}

func containsRole(id uuid.UUID, roles []data.Role) bool {
	for _, role := range roles {
		if id == role.ID {
			return true
		}
	}
	return false
}

func cleanRoles(store *bolthold.Store, tx *bolt.Tx, userID uuid.UUID, newRoles []data.Role) error {
	// get the current set of roles
	roles, err := userRoles(store, tx, userID)
	if err != nil {
		return err
	}

	var targets []Role
	for _, role := range roles {
		if !containsRole(role.ID, newRoles) {
			targets = append(targets, role)
		}
	}

	for _, target := range targets {
		if err := store.TxDelete(tx, target.ID, target); err != nil {
			return err
		}
	}

	return nil
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

func orgRoleUsers(store *bolthold.Store, tx *bolt.Tx, orgID uuid.UUID) ([]data.User, error) {
	roles, err := orgRoles(store, tx, orgID)
	if err != nil {
		return []data.User{}, err
	}

	return roleUsersData(store, tx, roles)
}

func orgRoles(store *bolthold.Store, tx *bolt.Tx, orgID uuid.UUID) (roles []Role, err error) {
	err = store.TxFind(tx, &roles, bolthold.Where("OrgID").Eq(orgID))
	return
}

func roleUsersData(store *bolthold.Store, tx *bolt.Tx, roles []Role) (users []data.User, err error) {
	userIDs := make(map[uuid.UUID]struct{})
	for _, role := range roles {
		userIDs[role.UserID] = struct{}{}
	}

	for id := range userIDs {
		var user User
		if err := store.TxGet(tx, id, &user); err != nil {
			return users, fmt.Errorf("problem finding user %q: %v", id, err)
		}
		users = append(users, data.User{
			ID:        user.ID,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Email:     user.Email,
			Pass:      user.Pass,
		})
	}
	return users, err
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
