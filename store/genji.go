package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"path"
	"sort"
	"sync"
	"time"

	"github.com/genjidb/genji"
	"github.com/genjidb/genji/document"
	genjierrors "github.com/genjidb/genji/errors"
	"github.com/genjidb/genji/types"
	"github.com/google/uuid"
	"github.com/simpleiot/simpleiot/data"
)

// Type defines the backing store used for the DB
type Type string

// define valid store types
const (
	StoreTypeMemory Type = "memory"
	StoreTypeBolt        = "bolt"
	StoreTypeBadger      = "badger"
)

// Meta contains metadata about the database
type Meta struct {
	ID      int    `json:"id"`
	Version int    `json:"version"`
	RootID  string `json:"rootID"`
}

// This file contains database manipulations.

// Db represents the store for the application
// FIXME, Db should be made private at some point
// and make the dump and import functions part of
// the store type.
type Db struct {
	store *genji.DB
	meta  Meta
	lock  sync.RWMutex

	// when the following cache data structures are accessed, you must take
	// the above lock
	nodeCache map[string]*data.Node
	edgeCache map[string]*data.Edge
}

// NewDb creates a new Db instance for the app
func NewDb(storeType Type, dataDir string) (*Db, error) {

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
		log.Fatal("Badger not currently supported")
		/*
			 // uncomment the following to enable badger support
				// Create a badger engine
				dbPath := path.Join(dataDir, "badger")
				ng, err := badgerengine.NewEngine(badger.DefaultOptions(dbPath))
				if err != nil {
					log.Fatal(err)
				}

				// Pass it to genji
				store, err = genji.New(context.Background(), ng)
		*/

	default:
		log.Fatal("Unknown store type: ", storeType)
	}

	err = store.Exec(`CREATE TABLE IF NOT EXISTS meta (id INT PRIMARY KEY)`)
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

	db := &Db{
		store:     store,
		nodeCache: make(map[string]*data.Node),
		edgeCache: make(map[string]*data.Edge),
	}
	return db, db.initialize()
}

// DBVersion for this version of siot
var DBVersion = 1

// initialize initializes the database metadata
func (gen *Db) initialize() error {
	// populate node cache
	err := gen.store.View(func(tx *genji.Tx) error {
		res, err := tx.Query(`select * from nodes`)
		if err != nil {
			return err
		}

		defer res.Close()

		gen.lock.Lock()
		defer gen.lock.Unlock()
		err = res.Iterate(func(d types.Document) error {
			var node data.Node
			err = document.StructScan(d, &node)
			if err != nil {
				return err
			}

			gen.nodeCache[node.ID] = &node
			return nil
		})

		return err
	})

	if err != nil {
		return err
	}

	// populate edge cache
	err = gen.store.View(func(tx *genji.Tx) error {
		res, err := tx.Query(`select * from edges`)
		if err != nil {
			return err
		}

		defer res.Close()

		gen.lock.Lock()
		defer gen.lock.Unlock()
		err = res.Iterate(func(d types.Document) error {
			var edge data.Edge
			err = document.StructScan(d, &edge)
			if err != nil {
				return err
			}

			gen.edgeCache[edge.ID] = &edge
			return nil
		})

		return err
	})

	if err != nil {
		return err
	}

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

	if err != genjierrors.ErrDocumentNotFound {
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

	if err != nil {
		return err
	}

	return err
}

// Close closes the db
func (gen *Db) Close() error {
	return gen.store.Close()
}

// rootNodeID returns the ID of the root node
func (gen *Db) rootNodeID() string {
	gen.lock.RLock()
	defer gen.lock.RUnlock()
	return gen.meta.RootID
}

func (gen *Db) node(id string) (*data.Node, error) {
	gen.lock.RLock()
	defer gen.lock.RUnlock()
	n, ok := gen.nodeCache[id]
	if !ok {
		return nil, data.ErrDocumentNotFound
	}
	// TODO do we need to make a copy here?
	return &(*n), nil
}

// recurisively find all descendents -- level is used to limit recursion
func (gen *Db) nodeFindDescendents(id string, recursive bool, level int) ([]data.NodeEdge, error) {
	var nodes []data.NodeEdge

	if level > 100 {
		return nodes, errors.New("Error: nodeFindDescendents, recursion limit reached")
	}

	edges := gen.edgeDown(id)

	for _, edge := range edges {
		node, err := gen.node(edge.Down)
		if err != nil {
			if err != genjierrors.ErrDocumentNotFound {
				// something bad happened
				return nodes, err
			}
			// else something is minorly wrong with db, print
			// error and return
			log.Println("Error finding node: ", edge.Down)
			continue
		}

		gen.lock.RLock()
		n := node.ToNodeEdge(*edge)
		gen.lock.RUnlock()

		nodes = append(nodes, n)

		tombstone, _ := n.IsTombstone()

		if recursive && !tombstone {
			downNodes, err := gen.nodeFindDescendents(edge.Down, true, level+1)
			if err != nil {
				return nodes, err
			}

			nodes = append(nodes, downNodes...)
		}
	}

	return nodes, nil
}

// nodeEdge returns a node edge
// parent can be:
//   - id of node
//   - none: parent details are skipped
//   - all: all parents are fetched
func (gen *Db) nodeEdge(id, parent string) ([]data.NodeEdge, error) {
	if id == "root" {
		id = gen.rootNodeID()
	}

	if parent == "" {
		parent = "none"
	}

	// get node
	node, err := gen.node(id)
	if err != nil {
		return []data.NodeEdge{}, err
	}

	if parent == "none" {
		// send back one node with null parent details
		hash, err := gen.calcHash(node, data.Edge{})
		if err != nil {
			return []data.NodeEdge{}, err
		}

		ret := node.ToNodeEdge(data.Edge{})
		ret.Hash = hash
		return []data.NodeEdge{ret}, nil
	}

	// find the edges and return multiple nodes
	e, err := gen.edgeUpDown(parent, id, true)

	if err != nil {
		return []data.NodeEdge{}, err
	}

	if len(e) <= 0 {
		return []data.NodeEdge{}, errors.New("No edges found")
	}

	ret := make([]data.NodeEdge, len(e))

	for i := 0; i < len(e); i++ {
		ret[i] = node.ToNodeEdge(e[i])
	}

	return ret, nil
}

// nodes returns all nodes.
func (gen *Db) nodes() ([]data.Node, error) {
	gen.lock.RLock()
	defer gen.lock.RUnlock()
	nodes := make([]data.Node, len(gen.nodeCache))

	i := 0
	for _, v := range gen.nodeCache {
		nodes[i] = *v
		i++
	}

	return nodes, nil
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

	current, _ := edge.Points.ValueBool(data.PointTypeTombstone, "")

	if current != tombstone {
		edge.Points.Add(data.Point{
			Type:  data.PointTypeTombstone,
			Value: data.BoolToFloat(tombstone),
			Time:  time.Now(),
		})

		sort.Sort(edge.Points)

		err := tx.Exec(`update edges set points = ? where id = ?`,
			edge.Points, edge.ID)

		if err != nil {
			return err
		}
	}

	return nil
}

var uuidZero uuid.UUID
var zero string

func init() {
	zero = uuidZero.String()
}

func (gen *Db) calcHash(node *data.Node, upEdge data.Edge) ([]byte, error) {
	// get child edges
	downEdges := gen.edgeDown(node.ID)

	upEdges := []*data.Edge{&upEdge}

	updateHash(node, upEdges, downEdges)

	return upEdge.Hash, nil
}

func (gen *Db) edgePoints(nodeID, parentID string, points data.Points) error {
	for _, p := range points {
		if p.Time.IsZero() {
			p.Time = time.Now()
		}
	}

	return gen.store.Update(func(tx *genji.Tx) error {
		if parentID == "none" && gen.meta.RootID != "" && nodeID != gen.meta.RootID {
			// a downstream node its root node edges, set up to rootID
			parentID = gen.meta.RootID
		}

		nec := newNodeEdgeCache(gen, tx)

		edge := &data.Edge{}
		newEdge := true

		gen.lock.RLock()
		for _, e := range gen.edgeCache {
			if e.Down == nodeID && e.Up == parentID {
				newEdge = false
				edge = e
				newEdge = false
				break
			}
		}
		gen.lock.RUnlock()

		if newEdge {
			edge.ID = uuid.New().String()
			edge.Up = parentID
			edge.Down = nodeID
			newEdge = true
		}

		nec.cacheEdges([]*data.Edge{edge})

		nodeExists := true
		ne, err := nec.getNodeAndEdges(edge.Down)
		if err != nil {
			// if this is a new node, the node may not exist
			// yet, which is OK
			if err != data.ErrDocumentNotFound {
				return fmt.Errorf("getNodeAndEdges error: %w", err)
			}

			nodeExists = false
		}

		if newEdge && nodeExists {
			ne.up = append(ne.up, edge)
		}

		gen.lock.Lock()
		for _, point := range points {
			edge.Points.Add(point)
		}
		gen.lock.Unlock()

		sort.Sort(edge.Points)

		if nodeExists {
			err = nec.processNode(ne, newEdge)
			if err != nil {
				return fmt.Errorf("processNode error: %w", err)
			}
		}

		err = nec.writeEdges()
		if err != nil {
			return err
		}

		return nil
	})
}

// nodePoints processes Points for a particular node
// this function does the following:
//   - updates the points in the node
//   - updates hash in all upstream edges
func (gen *Db) nodePoints(id string, points data.Points) error {
	for _, p := range points {
		if p.Time.IsZero() {
			p.Time = time.Now()
		}
	}

	return gen.store.Update(func(tx *genji.Tx) error {
		nec := newNodeEdgeCache(gen, tx)

		ne, err := nec.getNodeAndEdges(id)

		if err != nil {
			if err == data.ErrDocumentNotFound {
				if gen.meta.RootID == "" {
					gen.lock.Lock()
					gen.meta.RootID = id
					gen.lock.Unlock()
					err := tx.Exec(`update meta set rootid = ?`, id)
					if err != nil {
						return fmt.Errorf("Error setting rootid in meta: %w", err)
					}
				}

				ne = &nodeAndEdges{
					node: &data.Node{
						ID:   id,
						Type: data.NodeTypeDevice,
					},
				}

			} else {
				return err
			}
		}

		for _, point := range points {
			if point.Type == data.PointTypeNodeType {
				ne.node.Type = point.Text
				// we don't encode type in points as this has its own field
				continue
			}

			ne.node.Points.Add(point)
		}

		/*
			 * FIXME: need to clean up offline processing
			state := node.State()
			if state != data.PointValueSysStateOnline {
				node.Points.Add(
					data.Point{
						Time: time.Now(),
						Type: data.PointTypeSysState,
						Text: data.PointValueSysStateOnline,
					},
				)
			}
		*/

		sort.Sort(ne.node.Points)

		err = nec.processNode(ne, false)
		if err != nil {
			return fmt.Errorf("processNode error: %w", err)
		}

		err = nec.writeEdges()
		if err != nil {
			return err
		}

		gen.lock.Lock()
		gen.nodeCache[ne.node.ID] = ne.node
		gen.lock.Unlock()
		err = tx.Exec(`insert into nodes values ? on conflict do replace`, ne.node)

		if err != nil {
			return fmt.Errorf("Error inserting/updating node: %w", err)
		}

		return nil
	})
}

// nodeDescendents returns all descendents for a particular node ID and type
// set typ to blank string to find all descendents. Set recursive to false to
// stop at children, true to recursively get all descendents.
// FIXME, once recursion has been moved to client, this can return only a single
// level of []data.Node.
func (gen *Db) nodeDescendents(id, typ string, recursive, includeDel bool) ([]data.NodeEdge, error) {
	var nodes []data.NodeEdge

	childNodes, err := gen.nodeFindDescendents(id, recursive, 0)
	if err != nil {
		return []data.NodeEdge{}, err
	}

	for _, child := range childNodes {
		if !includeDel {
			tombstone, _ := child.IsTombstone()
			if tombstone {
				// skip deleted nodes
				continue
			}
		}
		if typ != "" {
			if child.Type == typ {
				nodes = append(nodes, child)
			}
		} else {
			nodes = append(nodes, child)
		}
	}

	return nodes, err
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

		err = res.Iterate(func(d types.Document) error {
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
func (gen *Db) edgeUp(nodeID string, includeTombstone bool) []*data.Edge {
	var ret []*data.Edge

	gen.lock.RLock()
	for _, e := range gen.edgeCache {
		if e.Down != nodeID {
			continue
		}

		if !e.IsTombstone() || includeTombstone {
			ret = append(ret, e)
		}
	}
	gen.lock.RUnlock()

	return ret
}

// find edge.
func (gen *Db) edgeUpDown(upID, downID string, includeTombstone bool) ([]data.Edge, error) {
	gen.lock.RLock()
	var ret []data.Edge
	defer gen.lock.RUnlock()
	for _, e := range gen.edgeCache {
		if (downID == "all" || e.Down == downID) && (upID == "all" || e.Up == upID) {
			if includeTombstone || !e.IsTombstone() {
				ret = append(ret, *e)
			}
		}
	}

	if len(ret) <= 0 {
		return nil, fmt.Errorf("Could not find edge, up: %v, down: %v", upID, downID)
	}

	return ret, nil
}

type downNode struct {
	id        string
	tombstone bool
}

// find downstream nodes
func (gen *Db) edgeDown(nodeID string) []*data.Edge {
	var ret []*data.Edge

	gen.lock.RLock()
	for _, e := range gen.edgeCache {
		if e.Up == nodeID {
			ret = append(ret, e)
		}
	}
	gen.lock.RUnlock()

	return ret
}

type privilege string

// minDistToRoot is used to calculate the minimum distance to the root node
func (gen *Db) minDistToRoot(id string) (int, error) {
	ret := 0
	var countUp func(string, int) (int, error)

	// recursive function to find the shortest distance to root node
	countUp = func(id string, count int) (int, error) {
		if gen.rootNodeID() == id {
			return count, nil
		}

		cnt := 10000000
		ups := gen.edgeUp(id, false)

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
		return 0, err
	}

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

// userCheck checks user authentication
// returns nil, nil if user is not found
func (gen *Db) userCheck(email, password string) (data.Nodes, error) {
	var ret []data.NodeEdge

	res, err := gen.store.Query(`select * from nodes where type = ?`, data.NodeTypeUser)
	if err != nil {
		// just return nil user and not user if not found
		if err == genjierrors.ErrDocumentNotFound {
			return nil, nil
		}

		return nil, err
	}
	defer res.Close()

	err = res.Iterate(func(d types.Document) error {
		var node data.Node
		err = document.StructScan(d, &node)
		if err != nil {
			return err
		}

		u := node.ToUser()

		if u.Email == email && u.Pass == password {
			edges := gen.edgeUp(node.ID, false)
			for _, edge := range edges {
				ne := node.ToNodeEdge(*edge)
				ret = append(ret, ne)
			}
		}

		return nil
	})

	return ret, err
}

type genImport struct {
	Nodes []data.Node `json:"nodes"`
	Edges []data.Edge `json:"edges"`
	Meta  Meta        `json:"meta"`
}

// ImportDb imports contents of file into database
func ImportDb(gen *Db, in io.Reader) error {
	decoder := json.NewDecoder(in)
	dump := genImport{}

	err := decoder.Decode(&dump)
	if err != nil {
		return err
	}

	// FIXME, re-import meta?
	return gen.store.Update(func(tx *genji.Tx) error {
		for _, n := range dump.Nodes {
			err := tx.Exec(`insert into nodes values ? on conflict do replace`, n)
			if err != nil {
				return fmt.Errorf("Error inserting node (%+v): %w", n, err)
			}
		}

		for _, e := range dump.Edges {
			err := tx.Exec(`insert into edges values ? on conflict do replace`, e)
			if err != nil {
				return fmt.Errorf("Error inserting edge (%+v): %w", e, err)
			}
		}

		if dump.Meta.RootID != "" {
			err := tx.Exec(`insert into meta values ? on conflict do replace`, dump.Meta)
			if err != nil {
				return fmt.Errorf("Error inserting meta (%+v): %w", dump.Meta, err)
			}
		}

		return nil
	})
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

	dump.Nodes, err = gen.nodes()
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
