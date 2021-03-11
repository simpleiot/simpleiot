package genji

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"path"
	"strings"
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/genjidb/genji"
	"github.com/genjidb/genji/database"
	"github.com/genjidb/genji/document"
	"github.com/genjidb/genji/engine/badgerengine"
	"github.com/google/uuid"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/db"
	bolt "go.etcd.io/bbolt"
)

// StoreType defines the backing store used for the DB
type StoreType string

// define valid store types
const (
	StoreTypeMemory StoreType = "memory"
	StoreTypeBolt             = "bolt"
	StoreTypeBadger           = "badger"
)

// Meta contains metadata about the database
type Meta struct {
	Version int
	RootID  string
}

// This file contains database manipulations.

// Db is used for all db access in the application.
// We will eventually turn this into an interface to
// handle multiple Db backends.
type Db struct {
	store  *genji.DB
	influx *db.Influx
	meta   Meta
}

// NewDb creates a new Db instance for the app
func NewDb(storeType StoreType, dataDir string, influx *db.Influx) (*Db, error) {

	var store *genji.DB
	var err error

	switch storeType {
	case StoreTypeMemory:
		store, err = genji.Open(":memory:")
		if err != nil {
			log.Fatal("Error opening memory store: ", err)
		}

	case StoreTypeBolt:
		dbFile := path.Join(dataDir, "data.db")
		store, err = genji.Open(dbFile)
		if err != nil {
			log.Fatal(err)
		}

	case StoreTypeBadger:
		// Create a badger engine
		dbPath := path.Join(dataDir, "badger")
		ng, err := badgerengine.NewEngine(badger.DefaultOptions(dbPath))
		if err != nil {
			log.Fatal(err)
		}

		// Pass it to genji
		store, err = genji.New(context.Background(), ng)

	default:
		log.Fatal("Unknown store type: ", storeType)
	}

	err = store.Exec(`CREATE TABLE IF NOT EXISTS meta`)
	if err != nil {
		return nil, fmt.Errorf("Error creating meta table: %w", err)
	}

	err = store.Exec(`CREATE TABLE IF NOT EXISTS nodes (id TEXT PRIMARY KEY)`)
	if err != nil {
		return nil, fmt.Errorf("Error creating nodes table: %w", err)
	}

	err = store.Exec(`CREATE INDEX IF NOT EXISTS idx_nodes_type ON nodes(type)`)
	if err != nil {
		return nil, fmt.Errorf("Error creating idx_nodes_type: %w", err)
	}

	err = store.Exec(`CREATE TABLE IF NOT EXISTS edges (id TEXT PRIMARY KEY)`)
	if err != nil {
		return nil, fmt.Errorf("Error creating edges table: %w", err)
	}

	err = store.Exec(`CREATE INDEX IF NOT EXISTS idx_edge_up ON edges(up)`)
	if err != nil {
		return nil, fmt.Errorf("Error creating idx_edge_up: %w", err)
	}

	err = store.Exec(`CREATE INDEX IF NOT EXISTS idx_edge_down ON edges(down)`)
	if err != nil {
		return nil, fmt.Errorf("Error creating idx_edge_down: %w", err)
	}

	err = store.Exec(`CREATE TABLE IF NOT EXISTS users (id TEXT PRIMARY KEY)`)
	if err != nil {
		return nil, fmt.Errorf("Error creating users table: %w", err)
	}

	err = store.Exec(`CREATE TABLE IF NOT EXISTS groups (id TEXT PRIMARY KEY)`)
	if err != nil {
		return nil, fmt.Errorf("Error creating groups table: %w", err)
	}

	err = store.Exec(`CREATE TABLE IF NOT EXISTS rules (id TEXT PRIMARY KEY)`)
	if err != nil {
		return nil, fmt.Errorf("Error creating rules table: %w", err)
	}

	err = store.Exec(`CREATE TABLE IF NOT EXISTS cmds (id TEXT PRIMARY KEY)`)
	if err != nil {
		return nil, fmt.Errorf("Error creating cmds table: %w", err)
	}

	db := &Db{store: store, influx: influx}
	return db, db.initialize()
}

// initialize initializes the database with one user (admin)
func (gen *Db) initialize() error {
	doc, err := gen.store.QueryDocument(`select * from meta`)

	// group was found or we ran into an error, so return
	if err == nil {
		// fetch metadata and return
		err := document.StructScan(doc, &gen.meta)
		if err != nil {
			return fmt.Errorf("Error getting db meta data: %w", err)
		}

		return nil
	}

	if err != database.ErrDocumentNotFound {
		return err
	}

	// need to initialize db
	err = gen.store.Update(func(tx *genji.Tx) error {
		log.Println("adding initial node structure and admin user ...")

		rootNode := data.Node{Type: data.NodeTypeDevice, ID: uuid.New().String()}

		err = tx.Exec(`insert into nodes values ?`, rootNode)
		if err != nil {
			return fmt.Errorf("Error creating root node: %w", err)
		}

		// populate metadata with root node ID
		gen.meta = Meta{RootID: rootNode.ID}

		err = tx.Exec(`insert into meta values ?`, gen.meta)
		if err != nil {
			return fmt.Errorf("Error inserting meta: %w", err)
		}

		// create admin user off root node
		admin := data.User{
			ID:        uuid.New().String(),
			FirstName: "admin",
			LastName:  "user",
			Email:     "admin@admin.com",
			Pass:      "admin",
		}

		adminUserNode := data.Node{
			Type:   data.NodeTypeUser,
			ID:     admin.ID,
			Points: admin.ToPoints()}

		err = tx.Exec(`insert into nodes values ?`, adminUserNode)
		if err != nil {
			return fmt.Errorf("Error inserting admin user: %w", err)
		}

		log.Println("Created admin user: ", admin)

		// create relationship between root and user node
		err = txEdgeInsert(tx, &data.Edge{Up: rootNode.ID, Down: adminUserNode.ID})
		if err != nil {
			return fmt.Errorf("Error creating root/admin edge: %w", err)
		}

		return nil
	})

	return err
}

// Close closes the db
func (gen *Db) Close() error {
	return gen.store.Close()
}

// RootNodeID returns the ID of the root node
func (gen *Db) RootNodeID() string {
	return gen.meta.RootID
}

func txNode(tx *genji.Tx, id string) (data.Node, error) {
	var node data.Node
	doc, err := tx.QueryDocument(`select * from nodes where id = ?`, id)
	if err != nil {
		return node, err
	}

	err = document.StructScan(doc, &node)
	return node, err
}

// recurisively find all descendents
func txNodeFindDescendents(tx *genji.Tx, id string, recursive bool) ([]data.NodeEdge, error) {
	var nodes []data.NodeEdge

	downIDs, err := txEdgeDown(tx, id)
	if err != nil {
		return nodes, err
	}

	for _, downID := range downIDs {
		node, err := txNode(tx, downID)
		if err != nil {
			if err != database.ErrDocumentNotFound {
				// something bad happened
				return nodes, err
			}
			// else something is minorly wrong with db, print
			// error and return
			log.Println("Error finding node: ", downID)
			continue
		}

		nodes = append(nodes, node.ToNodeEdge(id))

		if recursive {
			downNodes, err := txNodeFindDescendents(tx, downID, true)
			if err != nil {
				return nodes, err
			}

			nodes = append(nodes, downNodes...)
		}
	}

	return nodes, nil
}

// Node returns data for a particular node
func (gen *Db) Node(id string) (data.Node, error) {
	var node data.Node
	err := gen.store.View(func(tx *genji.Tx) error {
		var err error
		node, err = txNode(tx, id)
		return err
	})
	return node, err
}

func (gen *Db) txNodes(tx *genji.Tx) ([]data.Node, error) {
	var nodes []data.Node
	res, err := tx.Query(`select * from nodes`)
	if err != nil {
		return nodes, err
	}

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

// Nodes returns all nodes.
func (gen *Db) Nodes() ([]data.Node, error) {
	var nodes []data.Node

	err := gen.store.View(func(tx *genji.Tx) error {
		var err error
		nodes, err = gen.txNodes(tx)
		return err
	})

	return nodes, err
}

// NodeInsert is used to insert a node into the database
func (gen *Db) NodeInsert(node data.Node) (string, error) {
	if node.ID == "" {
		node.ID = uuid.New().String()
	}

	return node.ID, gen.store.Exec(`insert into nodes values ?`, node)
}

// NodeInsertEdge -- insert a node and edge and return uuid
// FIXME can we replace this with NATS calls?
func (gen *Db) NodeInsertEdge(node data.NodeEdge) (string, error) {
	if node.ID == "" {
		node.ID = uuid.New().String()
	}

	if node.Type == "" {
		return "", errors.New("New nodes must have a type")
	}

	err := gen.store.Update(func(tx *genji.Tx) error {
		err := tx.Exec(`insert into nodes values ?`, node.ToNode())
		if err != nil {
			return err
		}
		return txEdgeInsert(tx, &data.Edge{Up: node.Parent, Down: node.ID})
	})

	return node.ID, err
}

func txNodeDelete(tx *genji.Tx, id string) error {
	childIDs, err := txEdgeDown(tx, id)
	if err != nil {
		return err
	}

	for _, id := range childIDs {
		txNodeDelete(tx, id)
	}

	err = tx.Exec(`delete from nodes where id = ?`, id)
	if err != nil {
		return err
	}

	err = tx.Exec(`delete from edges where down = ?`, id)
	if err != nil {
		return err
	}

	err = tx.Exec(`delete from edges where up = ?`, id)
	if err != nil {
		return err
	}

	return nil
}

// NodeDelete deletes a node from the database and recursively all
// descendents
func (gen *Db) NodeDelete(id string) error {
	return gen.store.Update(func(tx *genji.Tx) error {
		return txNodeDelete(tx, id)
	})
}

var uuidZero uuid.UUID
var zero string

func init() {
	zero = uuidZero.String()
}

// NodePoint processes a Point for a particular node
func (gen *Db) NodePoint(id string, point data.Point) error {
	// for now, we process one point at a time. We may eventually
	// want to create NodeSamples to process multiple samples so
	// we can batch influx writes for performance

	if point.Time.IsZero() {
		point.Time = time.Now()
	}

	if gen.influx != nil {
		points := []db.InfluxPoint{
			db.PointToInfluxPoint(id, point),
		}
		err := gen.influx.WriteSamples(points)
		if err != nil {
			log.Println("Error writing points to influx: ", err)
		}
	}

	return gen.store.Update(func(tx *genji.Tx) error {
		var node data.Node
		doc, err := tx.QueryDocument(`select * from nodes where id = ?`, id)
		found := false

		if err != nil {
			if err == database.ErrDocumentNotFound {
				node.ID = id
				node.Type = data.NodeTypeDevice
			} else {
				return err
			}
		} else {
			err = document.StructScan(doc, &node)
			if err != nil {
				return err
			}
			found = true
		}

		node.ProcessPoint(point)
		node.SetState(data.PointValueSysStateOnline)

		if !found {
			err := tx.Exec(`insert into nodes values ?`, node)

			if err != nil {
				return err
			}

			return txEdgeInsert(tx, &data.Edge{
				Up: gen.meta.RootID, Down: id})
		}

		return tx.Exec(`update nodes set points = ? where id = ?`,
			node.Points, id)
	})
}

// NodeSetState is used to set the current system state
func (gen *Db) NodeSetState(id string, state string) error {
	return gen.store.Update(func(tx *genji.Tx) error {
		var node data.Node
		doc, err := tx.QueryDocument(`select * from nodes where id = ?`, id)
		if err != nil {
			return err
		}

		err = document.StructScan(doc, &node)
		if err != nil {
			return err
		}

		node.SetState(state)

		return tx.Exec(`update nodes set points = ? where id = ?`,
			node.Points, id)
	})
}

// NodeSetSwUpdateState is used to set the SW update state of the node
func (gen *Db) NodeSetSwUpdateState(id string, state data.SwUpdateState) error {
	return gen.store.Update(func(tx *genji.Tx) error {
		var node data.Node
		doc, err := tx.QueryDocument(`select * from nodes where id = ?`, id)
		if err != nil {
			return err
		}

		err = document.StructScan(doc, &node)
		if err != nil {
			return err
		}

		node.SetSwUpdateState(state)

		return tx.Exec(`update nodes set points = ? where id = ?`,
			node.Points, id)
	})
}

// NodeSetCmd sets a cmd for a node, and sets the
// CmdPending flag in the node structure.
func (gen *Db) NodeSetCmd(cmd data.NodeCmd) error {
	return gen.store.Update(func(tx *genji.Tx) error {
		// first update cmd table
		err := tx.Exec(`delete from cmds where id = ?`, cmd.ID)
		if err != nil {
			return err
		}

		err = tx.Exec(`insert into cmds values ?`, cmd)
		if err != nil {
			return err
		}

		// now update cmd pending in node
		doc, err := tx.QueryDocument(`select * from nodes where id = ?`, cmd.ID)
		if err != nil {
			return err
		}

		var node data.Node
		err = document.StructScan(doc, &node)
		if err != nil {
			return err
		}

		node.SetCmdPending(true)

		return tx.Exec(`update nodes set points = ? where id = ?`,
			node.Points, cmd.ID)
	})
}

// NodeDeleteCmd delets a cmd for a node and clears the
// the cmd pending flag
func (gen *Db) NodeDeleteCmd(id string) error {
	return gen.store.Update(func(tx *genji.Tx) error {
		err := tx.Exec(`delete from cmds where id = ?`, id)
		if err != nil {
			return err
		}

		// now update cmd pending in node
		doc, err := tx.QueryDocument(`select * from nodes where id = ?`, id)
		if err != nil {
			return err
		}

		var node data.Node
		err = document.StructScan(doc, &node)
		if err != nil {
			return err
		}

		node.SetCmdPending(false)

		return tx.Exec(`update nodes set points = ? where id = ?`,
			node.Points, id)
	})
}

// NodeGetCmd gets a cmd for a node. If the cmd is no null,
// the command is deleted, and the cmdPending flag cleared in
// the Node data structure.
func (gen *Db) NodeGetCmd(id string) (data.NodeCmd, error) {
	var cmd data.NodeCmd

	err := gen.store.Update(func(tx *genji.Tx) error {
		doc, err := tx.QueryDocument(`select * from cmds where id = ?`, id)

		if err != nil {
			if err != database.ErrDocumentNotFound {
				return err
			}
		}

		if err == nil {
			err = document.StructScan(doc, &cmd)
			if err != nil {
				return err
			}
		}

		if cmd.Cmd != "" {
			// a node has fetched a command, delete it
			err := tx.Exec(`delete from cmds where id = ?`, id)
			if err != nil {
				return err
			}

			// now update cmd pending in node
			doc, err := tx.QueryDocument(`select * from nodes where id = ?`, cmd.ID)
			if err != nil {
				return err
			}

			var node data.Node
			err = document.StructScan(doc, &node)
			if err != nil {
				return err
			}

			node.SetCmdPending(false)

			return tx.Exec(`update nodes set points = ? where id = ?`,
				node.Points, cmd.ID)

		}

		return nil
	})

	return cmd, err
}

// NodesForUser returns all nodes for a particular user
// FIXME this should be renamed to node children or something like that
func (gen *Db) NodesForUser(userID string) ([]data.NodeEdge, error) {
	var nodes []data.NodeEdge

	err := gen.store.View(func(tx *genji.Tx) error {
		// first find parents of user node
		rootNodeIDs, err := txEdgeUp(tx, userID)
		if err != nil {
			return err
		}

		if len(rootNodeIDs) == 0 {
			return errors.New("orphaned user")
		}

		for _, id := range rootNodeIDs {
			rootNode, err := txNode(tx, id)
			if err != nil {
				return err
			}
			nodes = append(nodes, rootNode.ToNodeEdge(""))

			childNodes, err := txNodeFindDescendents(tx, id, true)
			if err != nil {
				return err
			}

			nodes = append(nodes, childNodes...)
		}

		return nil
	})

	return nodes, err
}

// NodeDescendents returns all descendents for a particular node ID and type
// set typ to blank string to find all descendents. Set recursive to false to
// stop at children, true to recursively get all descendents.
func (gen *Db) NodeDescendents(id, typ string, recursive bool) ([]data.NodeEdge, error) {
	var nodes []data.NodeEdge

	err := gen.store.View(func(tx *genji.Tx) error {
		childNodes, err := txNodeFindDescendents(tx, id, recursive)
		if err != nil {
			return err
		}

		if typ == "" {
			nodes = append(nodes, childNodes...)
		} else {
			for _, child := range childNodes {
				if typ != "" {
					if child.Type == typ {
						nodes = append(nodes, child)
					}
				} else {
					nodes = append(nodes, child)
				}
			}
		}

		return nil
	})

	return nodes, err
}

func txEdgeInsert(tx *genji.Tx, edge *data.Edge) error {
	if edge.ID == "" {
		edge.ID = uuid.New().String()
	}

	return tx.Exec(`insert into edges values ?`, edge)
}

// EdgeInsert is used to insert an edge into the database
func (gen *Db) EdgeInsert(edge data.Edge) (string, error) {
	err := gen.store.Update(func(tx *genji.Tx) error {
		return txEdgeInsert(tx, &edge)
	})

	return edge.ID, err
}

// Edges returns all edges.
func (gen *Db) Edges() ([]data.Edge, error) {
	var edges []data.Edge

	err := gen.store.View(func(tx *genji.Tx) error {
		res, err := tx.Query(`select * from edges`)
		if err != nil {
			return err
		}

		defer res.Close()

		err = res.Iterate(func(d document.Document) error {
			var edge data.Edge
			err = document.StructScan(d, &edge)
			if err != nil {
				return err
			}

			edges = append(edges, edge)
			return nil
		})

		return nil
	})

	return edges, err
}

// find upstream nodes
func txEdgeUp(tx *genji.Tx, nodeID string) ([]string, error) {
	var ret []string
	res, err := tx.Query(`select * from edges where down = ?`, nodeID)
	if err != nil {
		return ret, err
	}
	defer res.Close()

	err = res.Iterate(func(d document.Document) error {
		var edge data.Edge
		err = document.StructScan(d, &edge)
		if err != nil {
			return err
		}

		ret = append(ret, edge.Up)
		return nil
	})

	return ret, err
}

// find downstream nodes
func txEdgeDown(tx *genji.Tx, nodeID string) ([]string, error) {
	var ret []string
	res, err := tx.Query(`select * from edges where up = ?`, nodeID)
	if err != nil {
		if err != database.ErrDocumentNotFound {
			return ret, err
		}

		return ret, nil
	}

	defer res.Close()

	err = res.Iterate(func(d document.Document) error {
		var edge data.Edge
		err = document.StructScan(d, &edge)
		if err != nil {
			return err
		}

		ret = append(ret, edge.Down)
		return nil
	})

	return ret, err
}

// EdgeUp returns an array of upstream nodes for a node
func (gen *Db) EdgeUp(nodeID string) ([]string, error) {
	var ret []string

	err := gen.store.View(func(tx *genji.Tx) error {
		var err error
		ret, err = txEdgeUp(tx, nodeID)
		return err
	})

	return ret, err
}

// EdgeMove is used to change a nodes parent
func (gen *Db) EdgeMove(id, oldParent, newParent string) error {
	return gen.store.Update(func(tx *genji.Tx) error {
		err := tx.Exec(`delete from edges where up = ? and down = ?`,
			oldParent, id)

		if err != nil {
			if err != database.ErrDocumentNotFound {
				return err
			}

			log.Println("Could not find old parent node: ", oldParent)
		}

		return txEdgeInsert(tx, &data.Edge{Up: newParent, Down: id})
	})
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
	var users []data.User
	res, err := gen.store.Query(`select * from users order by firstName`)
	if err != nil {
		return users, err
	}

	defer res.Close()

	err = res.Iterate(func(d document.Document) error {
		var user data.User
		err = document.StructScan(d, &user)
		if err != nil {
			return err
		}

		users = append(users, user)
		return nil
	})

	return users, err
}

type privilege string

// UserCheck checks user authentication
// returns nil, nil if user is not found
func (gen *Db) UserCheck(email, password string) (*data.User, error) {
	var user data.User

	res, err := gen.store.Query(`select * from nodes where type = ?`, data.NodeTypeUser)
	if err != nil {
		// just return nil user and not user if not found
		if err == database.ErrDocumentNotFound {
			return nil, nil
		}

		return nil, err
	}
	defer res.Close()

	err = res.Iterate(func(d document.Document) error {
		var node data.Node
		err = document.StructScan(d, &node)
		if err != nil {
			return err
		}

		u := node.ToUser()

		if u.Email == email && u.Pass == password {
			user = u
		}

		return nil
	})

	if user.ID != "" {
		return &user, err
	}

	return nil, err
}

// UserIsRoot checks if root user
func (gen *Db) UserIsRoot(id string) (bool, error) {
	upstreamNodes, err := gen.EdgeUp(id)
	if err != nil {
		return false, err
	}

	for _, upID := range upstreamNodes {
		if upID == gen.meta.RootID {
			return true, nil
		}
	}

	return false, nil
}

// UserByID returns the user with the given ID, if it exists.
func (gen *Db) UserByID(id string) (data.User, error) {
	var user data.User

	doc, err := gen.store.QueryDocument(`select * from users where id = ?`,
		id)

	if err != nil {
		return user, err
	}

	err = document.StructScan(doc, &user)
	return user, err
}

// UserByEmail returns the user with the given email, if it exists.
func (gen *Db) UserByEmail(email string) (data.User, error) {
	var user data.User

	doc, err := gen.store.QueryDocument(`select * from users where email = ?`,
		email)

	if err != nil {
		return user, err
	}

	err = document.StructScan(doc, &user)
	return user, err
}

// UsersForGroup returns all users who who are connected to a node by a group.
func (gen *Db) UsersForGroup(id string) ([]data.User, error) {
	var users []data.User

	err := gen.store.View(func(tx *genji.Tx) error {
		doc, err := tx.QueryDocument(`select * from groups where id = ?`, id)
		if err != nil {
			return err
		}

		var group data.Group
		err = document.StructScan(doc, &group)
		if err != nil {
			return err
		}

		for _, role := range group.Users {
			doc, err = tx.QueryDocument(`select * from users where id = ?`, role.UserID)
			if err != nil {
				return err
			}

			var user data.User
			err = document.StructScan(doc, &user)
			if err != nil {
				return err
			}
			users = append(users, user)
		}
		return nil
	})

	return users, err
}

// NodesForGroup returns the nodes which are property of the given Group.
func (gen *Db) NodesForGroup(tx *bolt.Tx, groupID string) ([]data.Node, error) {
	var nodes []data.Node
	res, err := gen.store.Query(`select * from nodes where ? in groups`,
		groupID)

	if err != nil {
		return nodes, err
	}

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
	if user.ID == "" {
		user.ID = uuid.New().String()
	}
	err := gen.store.Exec(`insert into users values ?`, user)
	return user.ID, err
}

// UserUpdate updates a new user
func (gen *Db) UserUpdate(user data.User) error {
	return gen.store.Update(func(tx *genji.Tx) error {
		err := gen.store.Exec(`delete from users where id = ?`,
			user.ID)

		if err != nil {
			if err != database.ErrDocumentNotFound {
				return err
			}
		}

		return gen.store.Exec(`insert into user values ?`, user)
	})
}

// UserDelete deletes a user from the database
func (gen *Db) UserDelete(id string) error {
	return gen.store.Exec(`delete from users where id = ?`, id)
}

func (gen *Db) txGroups(tx *genji.Tx) ([]data.Group, error) {
	var ret []data.Group

	res, err := tx.Query(`select * from groups`)
	if err != nil {
		return ret, err
	}

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

	return ret, err

}

// Groups returns all groups.
func (gen *Db) Groups() ([]data.Group, error) {
	var groups []data.Group

	err := gen.store.View(func(tx *genji.Tx) error {
		var err error
		groups, err = gen.txGroups(tx)
		return err
	})

	return groups, err
}

// Group returns the Group with the given ID.
func (gen *Db) Group(id string) (data.Group, error) {
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
	if group.ID == "" {
		group.ID = uuid.New().String()
	}
	if group.Parent == "" && group.ID != zero {
		group.Parent = zero
	}
	err := gen.store.Exec(`insert into groups values ?`, group)
	return group.ID, err
}

// GroupUpdate updates a group
func (gen *Db) GroupUpdate(gUpdate data.Group) error {
	return gen.store.Update(func(tx *genji.Tx) error {
		err := tx.Exec(`delete from groups where id = ?`, gUpdate.ID)
		if err != nil {
			if err != database.ErrDocumentNotFound {
				return err
			}
		}

		return tx.Exec(`insert into groups values ?`, gUpdate)
	})
}

// GroupDelete deletes a node from the database
func (gen *Db) GroupDelete(id string) error {
	return gen.store.Exec(`delete from groups where id = ?`, id)
}

// Rules returns all rules.
func (gen *Db) Rules() ([]data.Rule, error) {
	var ret []data.Rule
	res, err := gen.store.Query(`select * from rules`)
	if err != nil {
		return ret, err
	}

	defer res.Close()

	err = res.Iterate(func(d document.Document) error {
		var rule data.Rule
		err := document.StructScan(d, &rule)
		if err != nil {
			return err
		}

		ret = append(ret, rule)

		return nil
	})

	return ret, err
}

// RuleByID finds a rule given the ID
func (gen *Db) RuleByID(id string) (data.Rule, error) {
	var rule data.Rule
	doc, err := gen.store.QueryDocument(`select * from rules where id = ?`, id)
	if err != nil {
		return rule, err
	}

	err = document.StructScan(doc, &rule)
	return rule, err
}

// RuleInsert inserts a new rule
func (gen *Db) RuleInsert(rule data.Rule) (string, error) {
	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}
	err := gen.store.Update(func(tx *genji.Tx) error {
		err := tx.Exec(`insert into rules values ?`, rule)
		if err != nil {
			return err
		}

		doc, err := tx.QueryDocument(`select * from nodes where id = ?`,
			rule.Config.NodeID)

		if err != nil {
			return err
		}

		var node data.Node
		err = document.StructScan(doc, &node)
		if err != nil {
			return err
		}

		nodeHasRule := false

		for _, r := range node.Rules {
			if r == rule.ID {
				nodeHasRule = true
			}
		}

		if !nodeHasRule {
			node.Rules = append(node.Rules, rule.ID)

			return tx.Exec(`update nodes set rules = ? where id = ?`,
				node.Rules, node.ID)
		}

		return nil
	})

	return rule.ID, err
}

// RuleUpdateConfig updates a rule config
func (gen *Db) RuleUpdateConfig(id string, config data.RuleConfig) error {
	return gen.store.Exec(`update rules set config = ? where id = ?`,
		config, id)
}

// RuleUpdateState updates a rule state
func (gen *Db) RuleUpdateState(id string, state data.RuleState) error {
	return gen.store.Exec(`update rules set state = ? where id = ?`,
		state, id)
}

// RuleDelete deletes a rule from the database
func (gen *Db) RuleDelete(id string) error {
	return gen.store.Update(func(tx *genji.Tx) error {
		doc, err := tx.QueryDocument(`select * from rules where id = ?`,
			id)
		if err != nil {
			return err
		}

		var rule data.Rule
		err = document.StructScan(doc, &rule)
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
		newNodeRules := []string{}
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

// NodeCmds returns all cmds in database
func (gen *Db) NodeCmds() ([]data.NodeCmd, error) {
	var cmds []data.NodeCmd
	res, err := gen.store.Query(`select * from cmds`)
	if err != nil {
		return cmds, err
	}

	defer res.Close()

	err = res.Iterate(func(d document.Document) error {
		var cmd data.NodeCmd
		err = document.StructScan(d, &cmd)
		if err != nil {
			return err
		}

		cmds = append(cmds, cmd)
		return nil
	})

	return cmds, err
}

type genImport struct {
	Devices  []Device       `json:"devices"`
	Nodes    []data.Node    `json:"nodes"`
	Edges    []data.Edge    `json:"edges"`
	Users    []data.User    `json:"users"`
	Groups   []data.Group   `json:"groups"`
	Rules    []data.Rule    `json:"rules"`
	NodeCmds []data.NodeCmd `json:"nodeCmds"`
}

// ImportDb imports contents of file into database
func ImportDb(gen *Db, in io.Reader) error {
	decoder := json.NewDecoder(in)
	dump := genImport{}

	err := decoder.Decode(&dump)
	if err != nil {
		return err
	}

	for _, n := range dump.Nodes {
		_, err := gen.NodeInsert(n)
		if err != nil {
			return fmt.Errorf("Error inserting node (%+v): %w", n, err)
		}
	}

	for _, e := range dump.Edges {
		_, err := gen.EdgeInsert(e)
		if err != nil {
			return fmt.Errorf("Error inserting edge (%+v): %w", e, err)
		}
	}

	for _, d := range dump.Devices {
		n := d.ToNode()
		_, err := gen.NodeInsert(n)
		if err != nil {
			return fmt.Errorf("Error inserting node (%+v): %w", n, err)
		}
	}

	for _, u := range dump.Users {
		n := u.ToNode()
		ne := n.ToNodeEdge(gen.meta.RootID)
		_, err := gen.NodeInsertEdge(ne)
		if err != nil {
			return fmt.Errorf("Error inserting user (%+v): %w", u, err)
		}
	}

	for _, r := range dump.Rules {
		_, err := gen.RuleInsert(r)
		if err != nil {
			return fmt.Errorf("Error inserting rule (%+v): %w", r, err)
		}
	}

	return nil
}

type genDump struct {
	Nodes    []data.Node    `json:"nodes"`
	Edges    []data.Edge    `json:"edges"`
	Rules    []data.Rule    `json:"rules"`
	NodeCmds []data.NodeCmd `json:"nodeCmds"`
	Meta     Meta           `json:"meta"`
}

// DumpDb dumps the entire gen to a file
func DumpDb(gen *Db, out io.Writer) error {
	dump := genDump{}

	var err error

	dump.Nodes, err = gen.Nodes()
	if err != nil {
		return err
	}

	dump.Edges, err = gen.Edges()
	if err != nil {
		return err
	}

	dump.NodeCmds, err = gen.NodeCmds()
	if err != nil {
		return err
	}

	dump.Meta = gen.meta

	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "   ")

	return encoder.Encode(dump)
}
