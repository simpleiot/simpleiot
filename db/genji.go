package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"path"
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/genjidb/genji"
	"github.com/genjidb/genji/database"
	"github.com/genjidb/genji/document"
	"github.com/genjidb/genji/engine/badgerengine"
	"github.com/google/uuid"
	"github.com/simpleiot/simpleiot/data"
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
	store *genji.DB
	meta  Meta
}

// NewDb creates a new Db instance for the app
func NewDb(storeType StoreType, dataDir string) (*Db, error) {

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

	db := &Db{store: store}
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

func txNodeDelete(tx *genji.Tx, id, parent string) error {
	upIDs, err := txEdgeUp(tx, id)
	if err != nil {
		return err
	}

	err = tx.Exec(`delete from edges where down = ? and up = ?`, id, parent)
	if err != nil {
		return err
	}

	if len(upIDs) > 1 {
		// there are still other nodes using this node
		// so don't delete it
		return nil
	}

	// recursively delete all downstream nodes
	downIDs, err := txEdgeDown(tx, id)
	if err != nil {
		return err
	}

	for _, cid := range downIDs {
		txNodeDelete(tx, cid, id)
	}

	err = tx.Exec(`delete from nodes where id = ?`, id)
	if err != nil {
		return err
	}

	return nil
}

// NodeDelete deletes a node from the database and recursively all
// descendents
func (gen *Db) NodeDelete(id, parent string) error {
	return gen.store.Update(func(tx *genji.Tx) error {
		return txNodeDelete(tx, id, parent)
	})
}

var uuidZero uuid.UUID
var zero string

func init() {
	zero = uuidZero.String()
}

// NodePoint processes a Point for a particular node
func (gen *Db) nodePoint(id string, point data.Point) error {
	// for now, we process one point at a time. We may eventually
	// want to create NodePoints to process multiple points so
	// we can batch influx writes for performance

	if point.Time.IsZero() {
		point.Time = time.Now()
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

		hash := node.Points.ProcessPoint(point)
		state := node.State()
		if state != data.PointValueSysStateOnline {
			hash = node.Points.ProcessPoint(
				data.Point{
					Time: time.Now(),
					Type: data.PointTypeSysState,
					Text: data.PointValueSysStateOnline,
				},
			)
		}

		if !found {
			err := tx.Exec(`insert into nodes values ?`, node)

			if err != nil {
				return err
			}

			return txEdgeInsert(tx, &data.Edge{
				Up: gen.meta.RootID, Down: id})
		}

		return tx.Exec(`update nodes set points = ?, hash = ? where id = ?`,
			node.Points, hash, id)
	})
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

	return data.RemoveDuplicateNodesIDParent(nodes), err
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

// EdgeCopy is used to copy a node
func (gen *Db) EdgeCopy(id, newParent string) error {
	return gen.store.Update(func(tx *genji.Tx) error {
		return txEdgeInsert(tx, &data.Edge{Up: newParent, Down: id})
	})
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
// TODO: our current UI does not use the root user concept
// but we probably should implement something like that at some point
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

type genImport struct {
	Nodes []data.Node `json:"nodes"`
	Edges []data.Edge `json:"edges"`
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

	return nil
}

type genDump struct {
	Nodes []data.Node `json:"nodes"`
	Edges []data.Edge `json:"edges"`
	Meta  Meta        `json:"meta"`
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

	dump.Meta = gen.meta

	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "   ")

	return encoder.Encode(dump)
}
