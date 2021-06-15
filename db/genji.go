package db

import (
	"context"
	"crypto/md5"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"path"
	"sort"
	"sync"
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
	Version  int    `json:"version"`
	RootID   string `json:"rootID"`
	RootHash []byte `json:"rootHash"`
}

// This file contains database manipulations.

// Db is used for all db access in the application.
// We will eventually turn this into an interface to
// handle multiple Db backends.
type Db struct {
	store *genji.DB
	meta  Meta
	lock  sync.RWMutex
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

	/* Disable index for now as it is getting scrambled

	err = store.Exec(`CREATE INDEX IF NOT EXISTS idx_edge_up ON edges(up)`)
	if err != nil {
		return nil, fmt.Errorf("Error creating idx_edge_up: %w", err)
	}

	err = store.Exec(`CREATE INDEX IF NOT EXISTS idx_edge_down ON edges(down)`)
	if err != nil {
		return nil, fmt.Errorf("Error creating idx_edge_down: %w", err)
	}
	*/

	db := &Db{store: store}
	return db, db.initialize()
}

// DBVersion for this version of siot
var DBVersion = 1

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
		// populate metadata with root node ID
		gen.meta = Meta{Version: DBVersion}

		err = tx.Exec(`insert into meta values ?`, gen.meta)
		if err != nil {
			return fmt.Errorf("Error inserting meta: %w", err)
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
	gen.lock.RLock()
	defer gen.lock.RUnlock()
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

// recurisively find all descendents -- level is used to limit recursion
func txNodeFindDescendents(tx *genji.Tx, id string, recursive bool, level int) ([]data.NodeEdge, error) {
	var nodes []data.NodeEdge

	if level > 100 {
		return nodes, errors.New("Error: txNodeFindDescendents, recursion limit reached")
	}

	edges, err := txEdgeDown(tx, id)
	if err != nil {
		return nodes, err
	}

	for _, edge := range edges {
		node, err := txNode(tx, edge.Down)
		if err != nil {
			if err != database.ErrDocumentNotFound {
				// something bad happened
				return nodes, err
			}
			// else something is minorly wrong with db, print
			// error and return
			log.Println("Error finding node: ", edge.Down)
			continue
		}

		n := node.ToNodeEdge(edge)

		nodes = append(nodes, n)

		tombstone, _ := n.IsTombstone()

		if recursive && !tombstone {
			downNodes, err := txNodeFindDescendents(tx, edge.Down, true, level+1)
			if err != nil {
				return nodes, err
			}

			nodes = append(nodes, downNodes...)
		}
	}

	return nodes, nil
}

// FIXME, we should eventually remove this, or unexport it

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

// nodeEdge returns a node edge
func (gen *Db) nodeEdge(id, parent string) (data.NodeEdge, error) {
	var nodeEdge data.NodeEdge
	err := gen.store.View(func(tx *genji.Tx) error {
		node, err := txNode(tx, id)

		if err != nil {
			return err
		}

		if parent != "" {
			doc, err := tx.QueryDocument(`select * from edges where up = ? and down = ?`,
				parent, id)

			if err != nil {
				return err
			}

			var edge data.Edge

			err = document.StructScan(doc, &edge)

			if err != nil {
				return err
			}

			nodeEdge = node.ToNodeEdge(edge)
		} else {
			hash := []byte{}
			if gen.meta.RootID == id {
				hash = gen.meta.RootHash
			}
			nodeEdge = node.ToNodeEdge(data.Edge{Hash: hash})
		}

		return nil
	})

	return nodeEdge, err
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

// nodeInsert is used to insert a node into the database
func (gen *Db) nodeInsert(node data.Node) (string, error) {
	if node.ID == "" {
		node.ID = uuid.New().String()
	}

	return node.ID, gen.store.Exec(`insert into nodes values ?`, node)
}

func txSetTombstone(tx *genji.Tx, down, up string, tombstone bool) error {
	doc, err := tx.QueryDocument(`select * from edges where down = ? and up = ?`, down, up)
	if err != nil {
		return err
	}

	var edge data.Edge

	err = document.StructScan(doc, &edge)
	if err != nil {
		return err
	}

	current, _ := edge.Points.ValueBool("", data.PointTypeTombstone, 0)

	if current != tombstone {
		edge.Points.ProcessPoint(data.Point{
			Type:  data.PointTypeTombstone,
			Value: data.BoolToFloat(tombstone),
			Time:  time.Now(),
		})

		err := tx.Exec(`update edges set points = ? where id = ?`,
			edge.Points, edge.ID)

		if err != nil {
			return err
		}

	}

	return nil
}

func txNodeDelete(tx *genji.Tx, id, parent string) error {
	upIDs, err := txEdgeUp(tx, id)
	if err != nil {
		return err
	}

	err = txSetTombstone(tx, id, parent, true)

	if err != nil {
		return err
	}

	_ = upIDs

	/* for now we just leave all delete nodes in db -- we may change this later

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
	*/

	return nil
}

var uuidZero uuid.UUID
var zero string

func init() {
	zero = uuidZero.String()
}

func (gen *Db) nodeUpdateHash(id string) error {
	return gen.store.Update(func(tx *genji.Tx) error {
		node, err := txNode(tx, id)

		if err != nil {
			return err
		}

		isRoot := gen.meta.RootID == id

		// we need to update hash in all upstream paths from parent

		if !isRoot {
			upEdges, err := txEdgeUp(tx, id)

			if err != nil {
				return err
			}

			for _, upEdge := range upEdges {
				h := md5.New()

				for _, p := range upEdge.Points {
					d := make([]byte, 8)
					binary.LittleEndian.PutUint64(d, uint64(p.Time.UnixNano()))
					h.Write(d)
				}

				for _, p := range node.Points {
					d := make([]byte, 8)
					binary.LittleEndian.PutUint64(d, uint64(p.Time.UnixNano()))
					h.Write(d)
				}

				// get child edges
				downEdges, err := txEdgeDown(tx, id)

				if err != nil {
					return err
				}

				sort.Sort(data.ByEdgeID(downEdges))

				for _, downEdge := range downEdges {
					h.Write(downEdge.Hash)
				}

				// update hash in parent edge
				return tx.Exec(`update edges set hash = ? where id = ?`,
					h.Sum(nil), upEdge.ID)
			}
		} else {
			h := md5.New()

			for _, p := range node.Points {
				d := make([]byte, 8)
				binary.LittleEndian.PutUint64(d, uint64(p.Time.UnixNano()))
				h.Write(d)
			}

			// get child edges
			downEdges, err := txEdgeDown(tx, id)

			if err != nil {
				return err
			}

			sort.Sort(data.ByEdgeID(downEdges))

			for _, downEdge := range downEdges {
				h.Write(downEdge.Hash)
			}

			// update hash in parent edge
			return tx.Exec(`update meta set roothash = ?`,
				h.Sum(nil))
		}

		return nil
	})
}

func (gen *Db) edgePoints(id string, points data.Points) error {
	for _, p := range points {
		if p.Time.IsZero() {
			p.Time = time.Now()
		}
	}

	return gen.store.Update(func(tx *genji.Tx) error {
		var edge data.Edge

		doc, err := tx.QueryDocument(`select * from edges where id = ?`, id)

		if err != nil {
			return err
		}

		err = document.StructScan(doc, &edge)
		if err != nil {
			return err
		}

		for _, point := range points {
			edge.Points.ProcessPoint(point)
		}

		sort.Sort(edge.Points)

		err = tx.Exec(`update edges set points = ? where id = ?`,
			edge.Points, id)

		if err != nil {
			return err
		}

		return nil
	})

}

// nodePoints processes a Point for a particular node
func (gen *Db) nodePoints(id string, points data.Points) error {
	// for now, we process one point at a time. We may eventually
	// want to create NodePoints to process multiple points so
	// we can batch influx writes for performance

	for _, p := range points {
		if p.Time.IsZero() {
			p.Time = time.Now()
		}
	}

	return gen.store.Update(func(tx *genji.Tx) error {
		var node data.Node
		found := false

		doc, err := tx.QueryDocument(`select * from nodes where id = ?`, id)

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

		addParent := ""
		removeParent := ""

		for _, point := range points {
			if point.Type == data.PointTypeNodeType {
				node.Type = point.Text
				// we don't encode type in points as this has its own field
				continue
			}

			if point.Type == data.PointTypeAddParent {
				addParent = point.Text
				// we don't encode parent in points as this has its own field
				continue
			}

			if point.Type == data.PointTypeRemoveParent {
				removeParent = point.Text
				// we don't encode parent in points as this has its own field
				continue
			}

			node.Points.ProcessPoint(point)
		}

		state := node.State()
		if state != data.PointValueSysStateOnline {
			node.Points.ProcessPoint(
				data.Point{
					Time: time.Now(),
					Type: data.PointTypeSysState,
					Text: data.PointValueSysStateOnline,
				},
			)
		}

		sort.Sort(node.Points)

		if !found {
			err := tx.Exec(`insert into nodes values ?`, node)

			if err != nil {
				return err
			}

			if gen.meta.RootID == "" {
				log.Println("setting meta root id: ", node.ID)
				err := tx.Exec(`update meta set rootid = ?`, node.ID)
				if err != nil {
					return err
				}
				gen.lock.Lock()
				gen.meta.RootID = node.ID
				gen.lock.Unlock()
			} else {
				if addParent == "" {
					addParent = gen.meta.RootID
				}

				err := txEdgeInsert(tx, &data.Edge{
					Up: addParent, Down: id})

				if err != nil {
					return err
				}
			}
		} else {
			err := tx.Exec(`update nodes set points = ?, type = ? where id = ?`,
				node.Points, node.Type, id)

			if err != nil {
				return err
			}

			if addParent != "" {
				// make sure edge node to parent exists
				err := txEdgeInsert(tx, &data.Edge{
					Up: addParent, Down: id})

				if err != nil {
					return err
				}
			}
		}

		if removeParent != "" {
			err := txNodeDelete(tx, id, removeParent)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// NodesForUser returns all nodes for a particular user
// FIXME this should be renamed to node children or something like that
func (gen *Db) NodesForUser(userID string) ([]data.NodeEdge, error) {
	var nodes []data.NodeEdge

	err := gen.store.View(func(tx *genji.Tx) error {
		// first find parents of user node
		edges, err := txEdgeUp(tx, userID)
		if err != nil {
			return err
		}

		if len(edges) == 0 {
			return errors.New("orphaned user")
		}

		for _, edge := range edges {
			rootNode, err := txNode(tx, edge.Up)
			if err != nil {
				return err
			}
			nodes = append(nodes, rootNode.ToNodeEdge(data.Edge{}))

			childNodes, err := txNodeFindDescendents(tx, rootNode.ID, true, 0)
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
// FIXME, once recursion has been moved to client, this can return only a single
// level of []data.Node.
func (gen *Db) NodeDescendents(id, typ string, recursive bool) ([]data.NodeEdge, error) {
	var nodes []data.NodeEdge

	err := gen.store.View(func(tx *genji.Tx) error {
		childNodes, err := txNodeFindDescendents(tx, id, recursive, 0)
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
	_, err := tx.QueryDocument(`select * from edges where up = ? and down = ?`,
		edge.Up, edge.Down)

	if err != nil {
		if err == database.ErrDocumentNotFound {
			// create the edge entry
			if edge.ID == "" {
				edge.ID = uuid.New().String()
			}

			return tx.Exec(`insert into edges values ?`, edge)
		}

		return err
	}

	// edge already exists, make sure tombstone field is false
	return txSetTombstone(tx, edge.Down, edge.Up, false)
}

// edgeInsert is used to insert an edge into the database
func (gen *Db) edgeInsert(edge data.Edge) (string, error) {
	err := gen.store.Update(func(tx *genji.Tx) error {
		return txEdgeInsert(tx, &edge)
	})

	return edge.ID, err
}

// edges returns all edges.
func (gen *Db) edges() ([]data.Edge, error) {
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

// find upstream nodes. Does not include tombstoned edges.
func txEdgeUp(tx *genji.Tx, nodeID string) ([]data.Edge, error) {
	var ret []data.Edge
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

		if !edge.IsTombstone() {
			ret = append(ret, edge)
		}
		return nil
	})

	return ret, err
}

type downNode struct {
	id        string
	tombstone bool
}

// find downstream nodes
func txEdgeDown(tx *genji.Tx, nodeID string) ([]data.Edge, error) {
	var ret []data.Edge
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

		ret = append(ret, edge)
		return nil
	})

	return ret, err
}

// EdgeUp returns an array of upstream nodes for a node. Does not include
// tombstoned edges.
func (gen *Db) EdgeUp(nodeID string) ([]data.Edge, error) {
	var ret []data.Edge

	err := gen.store.View(func(tx *genji.Tx) error {
		var err error
		ret, err = txEdgeUp(tx, nodeID)
		return err
	})

	return ret, err
}

type privilege string

// minDistToRoot is used to calculate the minimum distance to the root node
func (gen *Db) minDistToRoot(id string) (int, error) {
	ret := 0
	err := gen.store.View(func(tx *genji.Tx) error {
		var countUp func(string, int) (int, error)

		// recursive function to find the shortest distance to root node
		countUp = func(id string, count int) (int, error) {
			if gen.RootNodeID() == id {
				return count, nil
			}

			cnt := 10000000
			ups, err := txEdgeUp(tx, id)
			if err != nil {
				return count, err
			}

			for _, up := range ups {
				c, err := countUp(up.Up, count+1)
				if err != nil {
					return count, err
				}
				if c < cnt {
					cnt = c
				}
			}

			return cnt, nil
		}

		var err error
		ret, err = countUp(id, 0)
		if err != nil {
			return err
		}

		return nil
	})

	return ret, err
}

type userDistRoot struct {
	distRoot int
	user     data.User
}

// we want to use the one closest to the root node for authentication
type byDistRoot []userDistRoot

// implement sort interface
func (b byDistRoot) Len() int           { return len(b) }
func (b byDistRoot) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byDistRoot) Less(i, j int) bool { return b[i].distRoot < b[j].distRoot }

// UserCheck checks user authentication
// returns nil, nil if user is not found
func (gen *Db) UserCheck(email, password string) (*data.User, error) {
	var users []userDistRoot

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
			distRoot, err := gen.minDistToRoot(u.ID)
			if err != nil {
				log.Println("Error getting dist to root: ", err)
			}
			users = append(users, userDistRoot{distRoot, u})
		}

		return nil
	})

	if len(users) > 0 {
		sort.Sort(byDistRoot(users))
		return &users[0].user, err
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

	for _, up := range upstreamNodes {
		if up.Up == gen.meta.RootID {
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
		_, err := gen.nodeInsert(n)
		if err != nil {
			return fmt.Errorf("Error inserting node (%+v): %w", n, err)
		}
	}

	for _, e := range dump.Edges {
		_, err := gen.edgeInsert(e)
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
		return fmt.Errorf("Error getting nodes: %v", err)
	}

	dump.Edges, err = gen.edges()
	if err != nil {
		return fmt.Errorf("Error getting edges: %v", err)
	}

	dump.Meta = gen.meta

	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "   ")

	err = encoder.Encode(dump)

	if err != nil {
		return fmt.Errorf("Error encoding: %v", err)
	}

	return nil
}
