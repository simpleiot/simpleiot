package genji

import (
	"encoding/json"
	"io"
	"log"
	"path"
	"sort"
	"strings"

	"github.com/genjidb/genji"
	"github.com/genjidb/genji/database"
	"github.com/genjidb/genji/document"
	"github.com/google/uuid"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/db"
	"github.com/timshannon/bolthold"
	bolt "go.etcd.io/bbolt"
)

// This file contains database manipulations.

// Db is used for all db access in the application.
// We will eventually turn this into an interface to
// handle multiple Db backends.
type Db struct {
	store  *genji.DB
	influx *db.Influx
}

// NewDb creates a new Db instance for the app
func NewDb(dataDir string, influx *db.Influx, init bool) (*Db, error) {
	dbFile := path.Join(dataDir, "data.db")

	store, err := genji.Open(dbFile)
	if err != nil {
		log.Fatal(err)
	}

	err = store.Exec("CREATE TABLE IF NOT EXISTS points;")
	if err != nil {
		return nil, err
	}

	err = store.Exec("CREATE TABLE IF NOT EXISTS users;")
	if err != nil {
		return nil, err
	}

	err = store.Exec("CREATE TABLE IF NOT EXISTS groups;")
	if err != nil {
		return nil, err
	}

	err = store.Exec("CREATE TABLE IF NOT EXISTS rules;")
	if err != nil {
		return nil, err
	}

	db := &Db{store: store, influx: influx}
	if init {
		return db, db.initialize()
	}

	return db, nil
}

// Close closes the db
func (gen *Db) Close() error {
	return gen.store.Close()
}

func (gen *Db) update(fn func(tx *bolt.Tx) error) error {
	return gen.store.Bolt().Update(fn)
}

func (gen *Db) view(fn func(tx *bolt.Tx) error) error {
	return gen.store.Bolt().View(fn)
}

// Node returns data for a particular device
func (gen *Db) Node(id string) (ret data.Node, err error) {
	err = gen.store.Get(id, &ret)
	return
}

// Nodes returns all devices.
func (gen *Db) Nodes() (ret []data.Node, err error) {
	err = gen.store.Find(&ret, nil)
	return
}

// NodeByID returns a device for a given ID
func (gen *Db) NodeByID(id string) (data.Node, error) {
	var ret data.Node
	if err := gen.store.Get(id, &ret); err != nil {
		return ret, err
	}

	return ret, nil
}

// NodeEach iterates through each device calling provided function
func (gen *Db) NodeEach(callback func(device *data.Node) error) error {
	return gen.store.ForEach(nil, callback)
}

// NodeDelete deletes a device from the database
func (gen *Db) NodeDelete(id string) error {
	return gen.update(func(tx *bolt.Tx) error {
		// first delete all rules for device
		var device data.Node
		err := gen.store.TxGet(tx, id, &device)
		if err != nil {
			return err
		}

		for _, r := range device.Rules {
			err := gen.store.TxDelete(tx, r.ID, data.Rule{})
			if err != nil {
				return err
			}
		}
		return gen.store.TxDelete(tx, id, data.Node{})
	})
}

// NodeUpdateGroups updates the groups for a device.
func (gen *Db) NodeUpdateGroups(id string, groups []uuid.UUID) error {
	return gen.update(func(tx *bolt.Tx) error {
		var dev data.Node
		if err := gen.store.TxGet(tx, id, &dev); err != nil {
			return err
		}

		dev.Groups = groups

		return gen.store.TxUpdate(tx, id, dev)
	})
}

var zero uuid.UUID

// NodePoint processes a Point for a particular device
func (gen *Db) NodePoint(id string, point data.Point) error {
	// for now, we process one point at a time. We may eventually
	// want to create NodeSamples to process multiple samples so
	// we can batch influx writes for performance

	if gen.influx != nil {
		points := []db.InfluxPoint{
			db.PointToInfluxPoint(id, point),
		}
		err := gen.influx.WriteSamples(points)
		if err != nil {
			log.Println("Error writing particle samples to influx: ", err)
		}
	}

	return gen.update(func(tx *bolt.Tx) error {
		var dev data.Node
		err := gen.store.TxGet(tx, id, &dev)
		if err != nil {
			if err == bolthold.ErrNotFound {
				dev.ID = id
			} else {
				return err
			}
		}

		dev.ProcessPoint(point)
		dev.SetState(data.SysStateOnline)
		return gen.store.TxUpsert(tx, id, dev)
	})
}

// NodeSetState is used to set the current system state
func (gen *Db) NodeSetState(id string, state int) error {
	return gen.update(func(tx *bolt.Tx) error {
		var dev data.Node
		err := gen.store.TxGet(tx, id, &dev)
		if err != nil {
			if err == bolthold.ErrNotFound {
				dev.ID = id
			} else {
				return err
			}
		}

		dev.SetState(state)
		return gen.store.TxUpsert(tx, id, dev)
	})
}

// NodeSetSwUpdateState is used to set the SW update state of the device
func (gen *Db) NodeSetSwUpdateState(id string, state data.SwUpdateState) error {
	return gen.update(func(tx *bolt.Tx) error {
		var dev data.Node
		err := gen.store.TxGet(tx, id, &dev)
		if err != nil {
			if err == bolthold.ErrNotFound {
				dev.ID = id
			} else {
				return err
			}
		}

		dev.SetSwUpdateState(state)
		return gen.store.TxUpsert(tx, id, dev)
	})
}

// NodeSetCmd sets a cmd for a device, and sets the
// CmdPending flag in the device structure.
func (gen *Db) NodeSetCmd(cmd data.NodeCmd) error {
	return gen.update(func(tx *bolt.Tx) error {
		err := gen.store.TxUpsert(tx, cmd.ID, &cmd)
		if err != nil {
			return err
		}

		// and set the device pending flag
		var dev data.Node
		err = gen.store.TxGet(tx, cmd.ID, &dev)
		if err != nil {
			return err
		}

		dev.SetCmdPending(true)
		return gen.store.TxUpdate(tx, cmd.ID, dev)
	})
}

// NodeDeleteCmd delets a cmd for a device and clears the
// the cmd pending flag
func (gen *Db) NodeDeleteCmd(id string) error {
	return gen.update(func(tx *bolt.Tx) error {
		err := gen.store.TxDelete(tx, id, data.NodeCmd{})
		if err != nil {
			return err
		}

		// and clear the device pending flag
		var dev data.Node
		err = gen.store.TxGet(tx, id, &dev)
		if err != nil {
			return err
		}

		dev.SetCmdPending(false)
		err = gen.store.TxUpdate(tx, id, dev)
		if err != nil {
			return err
		}

		return nil
	})
}

// NodeGetCmd gets a cmd for a device. If the cmd is no null,
// the command is deleted, and the cmdPending flag cleared in
// the Node data structure.
func (gen *Db) NodeGetCmd(id string) (data.NodeCmd, error) {
	var cmd data.NodeCmd

	err := gen.update(func(tx *bolt.Tx) error {
		err := gen.store.TxGet(tx, id, &cmd)
		if err == bolthold.ErrNotFound {
			// we don't consider this an error in this case
			err = nil
		}

		if err != nil {
			return err
		}

		if cmd.Cmd != "" {
			// a device has fetched a command, delete it
			err := gen.store.TxDelete(tx, id, data.NodeCmd{})
			if err != nil {
				return err
			}

			// and clear the device pending flag
			var dev data.Node
			err = gen.store.TxGet(tx, id, &dev)
			if err != nil {
				return err
			}

			dev.SetCmdPending(false)
			err = gen.store.TxUpdate(tx, id, dev)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return cmd, err
}

// NodeCmds returns all commands for device
func (gen *Db) NodeCmds() (ret []data.NodeCmd, err error) {
	err = gen.store.Find(&ret, nil)
	return
}

// NodesForUser returns all devices for a particular user
func (gen *Db) NodesForUser(userID uuid.UUID) ([]data.Node, error) {
	var devices []data.Node

	isRoot, err := gen.UserIsRoot(userID)
	if err != nil {
		return devices, err
	}

	if isRoot {
		// return all devices for root users
		err := gen.store.Find(&devices, nil)
		return devices, err
	}

	err = gen.view(func(tx *bolt.Tx) error {
		// First find groups users is part of
		var allGroups []data.Group
		err := gen.store.TxFind(tx, &allGroups, nil)

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
		err = gen.store.TxFind(tx, &devices,
			bolthold.Where("Groups").ContainsAny(bolthold.Slice(groupIDs)...))

		return nil
	})

	return devices, err
}

type users []data.User

func (u users) Len() int {
	return len(u)
}

func (u users) Less(i, j int) bool {
	return strings.ToLower((u)[i].FirstName) < strings.ToLower((u)[j].FirstName)
}

func (u users) Swap(i, j int) {
	u[i], u[j] = u[j], u[i]
}

// Users returns all users, sorted by first name.
func (gen *Db) Users() ([]data.User, error) {
	var ret users
	err := gen.store.Find(&ret, nil)
	// sort users by first name
	sort.Sort(ret)
	return ret, err
}

type privilege string

// UserCheck checks user authenticatino
func (gen *Db) UserCheck(email, password string) (*data.User, error) {
	var u data.User
	query := bolthold.Where("Email").Eq(email).
		And("Pass").Eq(password)
	err := gen.store.FindOne(&u, query)
	if err != nil {
		if err != bolthold.ErrNotFound {
			return nil, err
		}
		return nil, nil
	}

	return &u, nil
}

// UserIsRoot checks if root user
func (gen *Db) UserIsRoot(id uuid.UUID) (bool, error) {
	var group data.Group

	err := gen.store.FindOne(&group, bolthold.Where("ID").Eq(zero))

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

// UserByID returns the user with the given ID, if it exists.
func (gen *Db) UserByID(id string) (data.User, error) {
	var ret data.User
	if err := gen.store.FindOne(&ret, bolthold.Where("ID").Eq(id)); err != nil {
		return ret, err
	}

	return ret, nil
}

// UserByEmail returns the user with the given email, if it exists.
func (gen *Db) UserByEmail(email string) (data.User, error) {
	var ret data.User
	if err := gen.store.FindOne(&ret, bolthold.Where("Email").Eq(email)); err != nil {
		return ret, err
	}

	return ret, nil
}

// UsersForGroup returns all users who who are connected to a device by a group.
func (gen *Db) UsersForGroup(id uuid.UUID) ([]data.User, error) {
	var ret []data.User

	err := gen.view(func(tx *bolt.Tx) error {
		var group data.Group
		err := gen.store.TxGet(tx, id, &group)
		if err != nil {
			return err
		}

		for _, role := range group.Users {
			var user data.User
			err := gen.store.TxGet(tx, role.UserID, &user)
			if err != nil {
				return err
			}
			ret = append(ret, user)
		}
		return nil
	})

	return ret, err
}

// initialize initializes the database with one user (admin)
// in one groupanization (root).
// All devices are properties of the root groupanization.
func (gen *Db) initialize() error {
	// initialize root group in new gen
	var group data.Group

	_, err := gen.store.QueryDocument("select * from groups where name = root")

	// group was found or we ran into an error, so return
	if err != database.ErrDocumentNotFound {
		return err
	}

	err = gen.store.Update(func(tx *genji.Tx) error {
		log.Println("adding root group and admin user ...")

		admin := data.User{
			ID:        zero,
			FirstName: "admin",
			LastName:  "user",
			Email:     "admin@admin.com",
			Pass:      "admin",
		}

		err = tx.Exec(`insert into users values ?`, admin)

		if err != nil {
			return err
		}

		log.Println("Created admin user: ", admin)

		group = data.Group{
			ID:   zero,
			Name: "root",
			Users: []data.UserRoles{
				{UserID: zero, Roles: []data.Role{data.RoleAdmin}},
			},
		}

		err = tx.Exec(`insert into groups values ?`, group)

		if err != nil {
			return err
		}

		log.Println("Created root group:", group)

		return nil
	})

	return err
}

// NodesForGroup returns the nodes which are property of the given Group.
func (gen *Db) NodesForGroup(tx *bolt.Tx, groupID uuid.UUID) ([]data.Node, error) {
	var nodes []data.Node
	res, err := gen.store.Query("select * from nodes where ? in groups",
		groupID)

	defer res.Close()

	err = res.Iterate(func(d document.Document) error {
		var node data.Node
		err = document.StructScan(d, &node)
		if err != nil {
			return err
		}

		nodes = append(nodes, node)

		return nil
	})

	return nodes, err
}

// UserInsert inserts a new user
func (gen *Db) UserInsert(user data.User) (string, error) {
	id := uuid.New()
	user.Id = id
	err := gen.store.Exec(`insert into user values ?`, user)
	return id.String, err
}

// UserUpdate updates a new user
func (gen *Db) UserUpdate(user data.User) error {
	return gen.store.Update(func(tx *genji.Tx) error {
		err := gen.store.Exec(`delete from users where id = ?`,
			user.id)
		if err != nil {
			return err
		}

		return gen.store.Exec(`insert into user values ?`, user)
	})
}

// UserDelete deletes a user from the database
func (gen *Db) UserDelete(id uuid.UUID) error {
	return gen.store.Exec(`delete from users where id = ?`, id)
}

// Groups returns all groups.
func (gen *Db) Groups() ([]data.Group, error) {
	var ret []data.Group

	res, err := gen.store.Query(`select * from groups`)
	defer res.Close()

	err = res.Iterate(func(d document.Document) error {
		var group data.Group
		err := document.StructScan(d, &group)
		if err != nil {
			return err
		}
		ret = append(ret, group)
		return nil
	})

	return ret, nil
}

// Group returns the Group with the given ID.
func (gen *Db) Group(id uuid.UUID) (data.Group, error) {
	var ret data.Group
	doc, err := gen.store.QueryDocument(`select * from groups where id = ?`,
		id)
	if err != nil {
		return ret, err
	}

	err = document.StructScan(doc, &ret)
	return ret, err
}

// GroupInsert inserts a new group
func (gen *Db) GroupInsert(group data.Group) (string, error) {
	id := uuid.New()
	group.Parent = zero
	group.ID = id
	err := gen.store.Exec(`insert into groups values ?`, group)
	return id.String(), err
}

// GroupUpdate updates a group
func (gen *Db) GroupUpdate(gUpdate data.Group) error {
	return gen.store.Update(func(tx *genji.Tx) error {
		err := tx.Exec(`delete from groups where id = ?`, gUpdate.ID)
		if err != nil {
			return err
		}

		return tx.Exec(`insert into groups values ?`, gUpdate)
	})
}

// GroupDelete deletes a device from the database
func (gen *Db) GroupDelete(id uuid.UUID) error {
	return gen.store.Exec(`delete from groups where id = ?`, id)
}

// Rules returns all rules.
func (gen *Db) Rules() ([]data.Rule, error) {
	var ret []data.Rule
	res, err := gen.store.Query(`select * from rules`)
	if err != nil {
		return ret, err
	}
	res.Close()
	err = res.Iterate(func(d document.Document) {
		var rule data.Rule
		err := d.StructScan(&rule)
		if err != nil {
			return err
		}

		ret = append(ret, rule)

		return nil
	})

	return ret, err
}

// RuleByID finds a rule given the ID
func (gen *Db) RuleByID(id uuid.UUID) (data.Rule, error) {
	var rule data.Rule
	err := gen.store.Query(`select * from rules where id = ?`, id)
	return rule, err
}

// RuleInsert inserts a new rule
func (gen *Db) RuleInsert(rule data.Rule) (uuid.UUID, error) {
	rule.ID = uuid.New()
	err := gen.store.Update(func(tx *genji.Tx) error {
		err := tx.Exec(`insert into rules values ?`, rule)
		if err != nil {
			return err
		}

		doc, err := tx.QueryDocument(`select * from nodes where id = ?`,
			rule.config.NodeID)

		if err != nil {
			return err
		}

		var node data.Node
		err = doc.StructScan(&node)
		if err != nil {
			return err
		}

		node.Rules = append(node.Rules, rule.ID)

		return tx.Exec(`update nodes set rules = ? where id = ?`,
			node.Rules, node.ID)
	})

	return rule.ID, err
}

// RuleUpdateConfig updates a rule config
func (gen *Db) RuleUpdateConfig(id uuid.UUID, config data.RuleConfig) error {
	return gen.store.Exec(`update rules set config = ? where id = ?`,
		config, id)
}

// RuleUpdateState updates a rule state
func (gen *Db) RuleUpdateState(id uuid.UUID, state data.RuleState) error {
	return gen.store.Exec(`update rules set state = ? where id = ?`,
		state, id)
}

// RuleDelete deletes a rule from the database
func (gen *Db) RuleDelete(id uuid.UUID) error {
	return gen.store.Update(func(tx *genji.Tx) error {
		doc, err := tx.QueryDocument(`select * from rules where id = ?`,
			id)
		if err != nil {
			return err
		}

		var rule data.Rule
		err = doc.StructScan(&rule)
		if err != nil {
			return err
		}

		// remove rule from node
		doc, err = tx.QueryDocument(`select * from nodes where id = ?`,
			rule.Config.NodeID)
		if err != nil {
			return err
		}

		var node data.Node
		err = document.StructScan(doc, &node)
		if err != nil {
			return err
		}

		// new rules array
		newNodeRules := []uuid.UUID{}
		for _, rID := range node.Rules {
			if rID != rule.ID {
				newNodeRules = append(newNodeRules, rID)
			}
		}

		err = tx.Exec(`update nodes set rules = ? where id = ?`,
			newNodeRules, node.ID)

		if err != nil {
			return nil
		}

		return tx.Exec(`delete from rules where id = ?`, id)
	})
}

// STOPPED

// RuleEach iterates through each rule calling provided function
func (gen *Db) RuleEach(callback func(rule *data.Rule) error) error {
	return gen.store.ForEach(nil, callback)
}

func newIfZero(id uuid.UUID) uuid.UUID {
	if id == zero {
		return uuid.New()
	}
	return id
}

type genDump struct {
	Nodes    []data.Node    `json:"devices"`
	Users    []data.User    `json:"users"`
	Groups   []data.Group   `json:"groups"`
	Rules    []data.Rule    `json:"rules"`
	NodeCmds []data.NodeCmd `json:"deviceCmds"`
}

// DumpDb dumps the entire gen to a file
func DumpDb(gen *Db, out io.Writer) error {
	dump := genDump{}

	var err error

	dump.Nodes, err = gen.Nodes()
	if err != nil {
		return err
	}

	dump.Users, err = gen.Users()
	if err != nil {
		return err
	}

	dump.Groups, err = gen.Groups()
	if err != nil {
		return err
	}

	dump.Rules, err = gen.Rules()
	if err != nil {
		return err
	}

	dump.NodeCmds, err = gen.NodeCmds()
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "   ")

	return encoder.Encode(dump)
}
