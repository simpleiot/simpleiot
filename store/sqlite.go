package store

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/simpleiot/simpleiot/data"

	// tell sql to use sqlite
	_ "modernc.org/sqlite"
)

// DbSqlite represents a SQLite data store
type DbSqlite struct {
	db        *sql.DB
	meta      Meta
	writeLock sync.Mutex
}

// Meta contains metadata about the database
type Meta struct {
	ID      int    `json:"id"`
	Version int    `json:"version"`
	RootID  string `json:"rootID"`
}

// NewSqliteDb creates a new Sqlite data store
func NewSqliteDb(dbFile string) (*DbSqlite, error) {
	ret := &DbSqlite{}

	pragmas := "_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(8000)&_pragma=journal_size_limit(100000000)"

	dbFileOptions := fmt.Sprintf("%s?%s", dbFile, pragmas)

	db, err := sql.Open("sqlite", dbFileOptions)
	if err != nil {
		return nil, err
	}

	// Note, code should run with the following set, which ensures we don't have any
	// nested db operations. Ideally, all DB operations should exit before the next one
	// starts. Cache rows in memory if necessary to make this happen.
	// db.SetMaxOpenConns(1)

	ret.db = db

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS meta (id INT NOT NULL PRIMARY KEY,
				version INT,
				root_id TEXT)`)
	if err != nil {
		return nil, fmt.Errorf("Error creating meta table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS edges (id TEXT NOT NULL PRIMARY KEY,
				up TEXT,
				down TEXT,
				hash INT)`)

	if err != nil {
		return nil, fmt.Errorf("Error creating edges table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS node_points (id TEXT NOT NULL PRIMARY KEY,
				node_id TEXT,
				type TEXT,
				key TEXT,
				time_s INT,
				time_ns INT,
				idx REAL,
				value REAL,
				text TEXT,
				data BLOB,
				tombstone INT,
				origin TEXT)`)

	if err != nil {
		return nil, fmt.Errorf("Error creating node_points table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS edge_points (id TEXT NOT NULL PRIMARY KEY,
				edge_id TEXT,
				type TEXT,
				key TEXT,
				time_s INT,
				time_ns INT,
				idx REAL,
				value REAL,
				text TEXT,
				data BLOB,
				tombstone INT,
				origin TEXT)`)

	if err != nil {
		return nil, fmt.Errorf("Error creating edge_points table: %v", err)
	}

	metaRows, err := db.Query("SELECT * from meta")
	if err != nil {
		return nil, fmt.Errorf("Error quering meta: %v", err)
	}
	defer metaRows.Close()

	for metaRows.Next() {
		err = metaRows.Scan(&ret.meta.ID, &ret.meta.Version, &ret.meta.RootID)
		if err != nil {
			return nil, fmt.Errorf("Error scanning meta row: %v", err)
		}
	}
	if err := metaRows.Close(); err != nil {
		return nil, err
	}

	if ret.meta.RootID == "" {
		// we need to initialize root node and user
		ret.meta.RootID, err = ret.initRoot()
		if err != nil {
			return nil, fmt.Errorf("Error initializing root node: %v", err)
		}
	}

	// make sure we find root ID
	_, err = ret.node(ret.meta.RootID)
	if err != nil {
		return nil, fmt.Errorf("db constructor can't fetch root node: %v", err)
	}

	return ret, nil
}

// reset the database by permanently wiping all data
func (sdb *DbSqlite) reset() error {
	var err error

	// truncate several tables
	tables := []string{"meta", "edges", "node_points", "edge_points"}
	for _, v := range tables {
		_, err = sdb.db.Exec(`DELETE FROM ` + v)
		if err != nil {
			return fmt.Errorf("Error truncating table: %v", err)
		}
	}

	// we need to initialize root node and user
	sdb.meta.RootID, err = sdb.initRoot()
	if err != nil {
		return fmt.Errorf("Error initializing root node: %v", err)
	}

	// make sure we find root ID
	_, err = sdb.node(sdb.meta.RootID)
	if err != nil {
		return fmt.Errorf("db constructor can't fetch root node: %v", err)
	}

	return nil
}

// verifyNodeHashes recursively verifies all the hash values for all nodes
func (sdb *DbSqlite) verifyNodeHashes() error {
	// must run this in a transaction so we don't get any modifications
	// while reading child nodes. This may be expensive for a large DB, so
	// we may want to eventually break this down into transactions for each node
	// and its children.
	tx, err := sdb.db.Begin()
	if err != nil {
		return err
	}

	rollback := func() {
		rbErr := tx.Rollback()
		if rbErr != nil {
			log.Println("Rollback error: ", rbErr)
		}
	}

	// get root node to kick things off
	rootNodes, err := sdb.getNodes(nil, "root", "all", "", true)

	if err != nil {
		rollback()
		return err
	}

	if len(rootNodes) < 1 {
		rollback()
		return errors.New("no root nodes")
	}

	root := rootNodes[0]

	var verify func(node data.NodeEdge) error

	verify = func(node data.NodeEdge) error {
		children, err := sdb.getNodes(nil, node.ID, "all", "", true)
		if err != nil {
			return err
		}

		hash := node.CalcHash(children)

		if hash != node.Hash {
			return fmt.Errorf("Hash failed for %v, stored: %v, calc: %v",
				node.ID, node.Hash, hash)
		}

		for _, c := range children {
			err := verify(c)
			if err != nil {
				return err
			}
		}

		return nil
	}

	err = verify(root)

	if err != nil {
		rollback()
		return fmt.Errorf("Verify failed: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (sdb *DbSqlite) initRoot() (string, error) {
	log.Println("STORE: Initialize root node and admin user")
	var rootNode data.NodeEdge
	rootNode.Points = data.Points{
		{
			Time: time.Now(),
			Type: data.PointTypeNodeType,
			Text: data.NodeTypeDevice,
		},
	}

	rootNode.ID = uuid.New().String()

	err := sdb.edgePoints(rootNode.ID, "root", data.Points{{Type: data.PointTypeTombstone, Value: 0}})
	if err != nil {
		return "", fmt.Errorf("Error sending root node edges: %w", err)
	}

	err = sdb.nodePoints(rootNode.ID, rootNode.Points)
	if err != nil {
		return "", fmt.Errorf("Error setting root node points: %v", err)
	}

	// create admin user off root node
	admin := data.User{
		ID:        uuid.New().String(),
		FirstName: "admin",
		LastName:  "user",
		Email:     "admin@admin.com",
		Pass:      "admin",
	}

	points := admin.ToPoints()

	err = sdb.edgePoints(admin.ID, rootNode.ID, data.Points{{Type: data.PointTypeTombstone, Value: 0}})
	if err != nil {
		return "", err
	}

	err = sdb.nodePoints(admin.ID, points)
	if err != nil {
		return "", fmt.Errorf("Error setting default user: %v", err)
	}

	sdb.writeLock.Lock()
	defer sdb.writeLock.Unlock()
	_, err = sdb.db.Exec("INSERT INTO meta(id, version, root_id) VALUES(?, ?, ?)", 0, 0, rootNode.ID)
	if err != nil {
		return "", fmt.Errorf("Error setting meta data: %v", err)
	}

	return rootNode.ID, nil
}

func (sdb *DbSqlite) nodePoints(id string, points data.Points) error {
	sdb.writeLock.Lock()
	defer sdb.writeLock.Unlock()
	tx, err := sdb.db.Begin()
	if err != nil {
		return err
	}

	rollback := func() {
		rbErr := tx.Rollback()
		if rbErr != nil {
			log.Println("Rollback error: ", rbErr)
		}
	}

	rowsPoints, err := tx.Query("SELECT * FROM node_points WHERE node_id=?", id)
	if err != nil {
		rollback()
		return err
	}
	defer rowsPoints.Close()

	var dbPoints data.Points
	var dbPointIDs []string

	for rowsPoints.Next() {
		var p data.Point
		var timeS, timeNS int64
		var pID string
		var nodeID string
		err := rowsPoints.Scan(&pID, &nodeID, &p.Type, &p.Key, &timeS, &timeNS, &p.Index, &p.Value, &p.Text,
			&p.Data, &p.Tombstone, &p.Origin)
		if err != nil {
			rollback()
			return err
		}
		p.Time = time.Unix(timeS, timeNS)
		dbPoints = append(dbPoints, p)
		dbPointIDs = append(dbPointIDs, pID)
	}

	if err := rowsPoints.Close(); err != nil {
		rollback()
		return fmt.Errorf("Error closing rowsPoints: %v", err)
	}

	var writePoints data.Points
	var writePointIDs []string

	var hashUpdate uint32

NextPin:
	for _, pIn := range points {
		if pIn.Time.IsZero() {
			pIn.Time = time.Now()
		}

		for j, pDb := range dbPoints {
			if pIn.Type == pDb.Type && pIn.Key == pDb.Key {
				// found a match
				if pDb.Time.Before(pIn.Time) || pDb.Time.Equal(pIn.Time) {
					writePoints = append(writePoints, pIn)
					writePointIDs = append(writePointIDs, dbPointIDs[j])
					// back out old CRC and add in new one
					hashUpdate ^= pDb.CRC()
					hashUpdate ^= pIn.CRC()
				} else {
					log.Println("Ignoring node point due to timestamps: ", id, pIn)
				}
				continue NextPin
			}
		}

		// point was not found so write it
		writePoints = append(writePoints, pIn)
		hashUpdate ^= pIn.CRC()
		writePointIDs = append(writePointIDs, uuid.New().String())
	}

	stmt, err := tx.Prepare(`INSERT INTO node_points(id, node_id, type, key, time_s,
                 time_ns, idx, value, text, data, tombstone, origin)
		 VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		 type = ?3,
		 key = ?4,
		 time_s = ?5,
		 time_ns = ?6,
		 idx = ?7,
		 value = ?8,
		 text = ?9,
		 data = ?10,
		 tombstone = ?11,
		 origin = ?12
		 `)
	defer stmt.Close()

	for i, p := range writePoints {
		tS := p.Time.Unix()
		tNs := p.Time.UnixNano() - 1e9*tS
		pID := writePointIDs[i]
		_, err = stmt.Exec(pID, id, p.Type, p.Key, tS, tNs, p.Index, p.Value, p.Text, p.Data, p.Tombstone,
			p.Origin)
		if err != nil {
			rollback()
			return err
		}
	}

	stmt.Close()

	err = sdb.updateHash(tx, id, hashUpdate)
	if err != nil {
		rollback()
		return fmt.Errorf("Error updating upstream hash: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (sdb *DbSqlite) edgePoints(nodeID, parentID string, points data.Points) error {
	sdb.writeLock.Lock()
	defer sdb.writeLock.Unlock()

	var err error
	if parentID == "" {
		parentID = "root"
	}

	tx, err := sdb.db.Begin()
	if err != nil {
		return err
	}

	rollback := func() {
		rbErr := tx.Rollback()
		if rbErr != nil {
			log.Println("Rollback error: ", rbErr)
		}
	}

	edges, err := sdb.edges(tx, "SELECT * FROM edges WHERE up=? AND down=?", parentID, nodeID)
	if err != nil {
		rollback()
		return err
	}

	var edge data.Edge

	newEdge := false

	if len(edges) <= 0 {
		newEdge = true
		edge.ID = uuid.New().String()
	} else {
		edge = edges[0]
	}

	rowsPoints, err := tx.Query("SELECT * FROM edge_points WHERE edge_id=?", edge.ID)
	if err != nil {
		rollback()
		return err
	}
	defer rowsPoints.Close()

	var dbPoints data.Points
	var dbPointIDs []string

	for rowsPoints.Next() {
		var p data.Point
		var timeS, timeNS int64
		var pID string
		var nodeID string
		err := rowsPoints.Scan(&pID, &nodeID, &p.Type, &p.Key, &timeS, &timeNS, &p.Index, &p.Value, &p.Text,
			&p.Data, &p.Tombstone, &p.Origin)
		if err != nil {
			rollback()
			return err
		}
		p.Time = time.Unix(timeS, timeNS)
		dbPoints = append(dbPoints, p)
		dbPointIDs = append(dbPointIDs, pID)
	}

	if err := rowsPoints.Close(); err != nil {
		rollback()
		return fmt.Errorf("Error closing rowsPoints: %v", err)
	}

	var writePoints data.Points
	var writePointIDs []string

	var hashUpdate uint32

NextPin:
	for _, pIn := range points {
		if pIn.Time.IsZero() {
			pIn.Time = time.Now()
		}

		for j, pDb := range dbPoints {
			if pIn.Type == pDb.Type && pIn.Key == pDb.Key {
				// found a match
				if pDb.Time.Before(pIn.Time) || pDb.Time.Equal(pIn.Time) {
					writePoints = append(writePoints, pIn)
					writePointIDs = append(writePointIDs, dbPointIDs[j])
					// back out old CRC and add in new one
					hashUpdate ^= pDb.CRC()
					hashUpdate ^= pIn.CRC()
				} else {
					log.Println("Ignoring edge point due to timestamps: ", edge.ID, pIn)
				}
				continue NextPin
			}
		}

		// point was not found so write it
		writePoints = append(writePoints, pIn)
		hashUpdate ^= pIn.CRC()
		writePointIDs = append(writePointIDs, uuid.New().String())
	}

	// loop through write points and write them
	stmt, err := tx.Prepare(`INSERT INTO edge_points(id, edge_id, type, key, time_s,
                 time_ns, idx, value, text, data, tombstone, origin)
		 VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		 type = ?3,
		 key = ?4,
		 time_s = ?5,
		 time_ns = ?6,
		 idx = ?7,
		 value = ?8,
		 text = ?9,
		 data = ?10,
		 tombstone = ?11,
		 origin = ?12
		 `)

	for i, p := range writePoints {
		tS := p.Time.Unix()
		tNs := p.Time.UnixNano() - 1e9*tS
		pID := writePointIDs[i]
		_, err = stmt.Exec(pID, edge.ID, p.Type, p.Key, tS, tNs, p.Index, p.Value, p.Text, p.Data, p.Tombstone,
			p.Origin)
		if err != nil {
			stmt.Close()
			rollback()
			return err
		}
	}

	stmt.Close()

	// we don't update the hash here as it gets updated later in updateHash()
	// SQLite is amazing as it appears the below INSERT can be read later in the read before
	// the transaction is finished.

	// write edge
	if newEdge {
		// did not find edge, need to add it
		edge.Up = parentID
		edge.Down = nodeID

		_, err := tx.Exec(`INSERT INTO edges(id, up, down, hash) VALUES (?, ?, ?, ?)`,
			edge.ID, edge.Up, edge.Down, edge.Hash)

		if err != nil {
			log.Println("edge insert failed, trying again ...: ", err)
			// FIXME, occasionaly the above INSERT will fail with "database is locked (5) (SQLITE_BUSY)"
			// FIXME, not sure if retry is required any more since we removed the nested
			// queries
			// not sure why, but the below retry seems to work around this issue for now
			_, err := tx.Exec(`INSERT INTO edges(id, up, down, hash) VALUES (?, ?, ?, ?)`,
				edge.ID, edge.Up, edge.Down, edge.Hash)

			// TODO check for downstream node and add in its hash

			if err != nil {
				rollback()
				return fmt.Errorf("Error when writing edge: %v", err)
			}
		}
	} else {
		// update hash
		_, err := tx.Exec(`UPDATE edges SET hash = ? WHERE id = ?`, edge.Hash, edge.ID)
		if err != nil {
			rollback()
			return fmt.Errorf("Error updating edge hash")
		}
	}

	// TODO: update upstream hash values
	err = sdb.updateHash(tx, nodeID, hashUpdate)
	if err != nil {
		rollback()
		return fmt.Errorf("Error updating upstream hash")
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (sdb *DbSqlite) updateHash(tx *sql.Tx, id string, hashUpdate uint32) error {
	// key in edgeCache is up-down
	cache := make(map[string]uint32)
	err := sdb.updateHashHelper(tx, id, hashUpdate, cache)
	if err != nil {
		return err
	}

	// write update hash values back to edges
	stmt, err := tx.Prepare(`UPDATE edges SET hash = ? WHERE id = ?`)

	for id, hash := range cache {
		_, err = stmt.Exec(hash, id)
		if err != nil {
			stmt.Close()
			return fmt.Errorf("Error updating edge hash: %v", err)
		}
	}

	stmt.Close()

	return nil
}

func (sdb *DbSqlite) updateHashHelper(tx *sql.Tx, id string, hashUpdate uint32, cache map[string]uint32) error {
	edges, err := sdb.edges(tx, "SELECT * FROM edges WHERE down=?", id)
	if err != nil {
		return err
	}

	for _, e := range edges {
		if _, ok := cache[e.ID]; !ok {
			cache[e.ID] = e.Hash
		}

		cache[e.ID] ^= hashUpdate

		if e.Up != "none" {
			err := sdb.updateHashHelper(tx, e.Up, hashUpdate, cache)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (sdb *DbSqlite) edges(tx *sql.Tx, query string, args ...any) ([]data.Edge, error) {
	var rowsEdges *sql.Rows
	var err error

	if tx != nil {
		rowsEdges, err = tx.Query(query, args...)
	} else {
		rowsEdges, err = sdb.db.Query(query, args...)
	}

	if err != nil {
		return nil, fmt.Errorf("Error getting edges: %v", err)
	}
	defer rowsEdges.Close()

	var edges []data.Edge

	for rowsEdges.Next() {
		var edge data.Edge
		err = rowsEdges.Scan(&edge.ID, &edge.Up, &edge.Down, &edge.Hash)
		if err != nil {
			return nil, fmt.Errorf("Error scanning edges: %v", err)
		}

		edges = append(edges, edge)
	}

	if err := rowsEdges.Close(); err != nil {
		return nil, err
	}

	return edges, nil
}

// Close the db
func (sdb *DbSqlite) Close() error {
	return sdb.db.Close()
}

func (sdb *DbSqlite) rootNodeID() string {
	return sdb.meta.RootID
}

// gets a node
func (sdb *DbSqlite) node(id string) (*data.Node, error) {

	var err error
	var ret data.Node
	ret.ID = id

	ret.Points, ret.Type, err = sdb.queryPoints(nil,
		"SELECT * FROM node_points WHERE node_id=?", id)

	if err != nil {
		return nil, err
	}

	if ret.Type == "" {
		return nil, data.ErrDocumentNotFound
	}

	return &ret, err
}

// If parent is set to "none", the edge details are not included
// and the hash is blank.
// If parent is set to "all", then all instances of the node are returned.
// If parent is set and id is "all", then all child nodes are returned.
// Parent can be set to "root" and id to "all" to fetch the root node(s).
func (sdb *DbSqlite) getNodes(tx *sql.Tx, parent, id, typ string, includeDel bool) ([]data.NodeEdge, error) {
	var ret []data.NodeEdge

	if parent == "" {
		parent = "none"
	}

	if id == "" {
		id = "all"
	}

	var q string

	switch {
	case parent == "none" && id == "all":
		return nil, errors.New("invalid combination of parent and id")
	case parent == "all" && id == "all":
		return nil, errors.New("invalid combination of parent and id")
	case parent == "none":
		node, err := sdb.node(id)
		if err != nil {
			return ret, err
		}
		ne := node.ToNodeEdge(data.Edge{})
		return []data.NodeEdge{ne}, nil
	case parent == "all":
		q = fmt.Sprintf("SELECT * FROM edges WHERE down = '%v'", id)
	case id == "all":
		q = fmt.Sprintf("SELECT * FROM edges WHERE up = '%v'", parent)
	default:
		// both parent and id are specified
		q = fmt.Sprintf("SELECT * FROM edges WHERE up='%v' AND down = '%v'", parent, id)
	}

	edges, err := sdb.edges(tx, q)

	if len(edges) < 1 {
		return ret, nil
	}

	for _, edge := range edges {
		var ne data.NodeEdge
		ne.ID = edge.Down
		ne.Parent = edge.Up
		ne.Hash = edge.Hash

		ne.EdgePoints, _, err = sdb.queryPoints(tx,
			"SELECT * FROM edge_points WHERE edge_id=?", edge.ID)
		if err != nil {
			return nil, fmt.Errorf("children error getting edge points: %v", err)
		}

		if !includeDel {
			tombstone, _ := ne.IsTombstone()
			if tombstone {
				// skip deleted nodes
				continue
			}
		}

		ne.Points, ne.Type, err = sdb.queryPoints(tx,
			"SELECT * FROM node_points WHERE node_id=?", edge.Down)
		if err != nil {
			return nil, fmt.Errorf("children error getting node points: %v", err)
		}

		if typ != "" {
			if ne.Type != typ {
				// skip node
				continue
			}
		}

		ret = append(ret, ne)
	}

	return ret, nil
}

// returns points, type (if node), and error
func (sdb *DbSqlite) queryPoints(tx *sql.Tx, query string, args ...any) (data.Points, string, error) {
	var retPoints data.Points
	var retType string
	rowsPoints, err := sdb.db.Query(query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rowsPoints.Close()

	for rowsPoints.Next() {
		var p data.Point
		var timeS, timeNS int64
		var pID string
		var nodeID string
		err := rowsPoints.Scan(&pID, &nodeID, &p.Type, &p.Key, &timeS, &timeNS, &p.Index, &p.Value, &p.Text,
			&p.Data, &p.Tombstone, &p.Origin)
		if err != nil {
			return nil, "", err
		}
		p.Time = time.Unix(timeS, timeNS)
		if p.Type == data.PointTypeNodeType {
			retType = p.Text
		} else {
			retPoints = append(retPoints, p)
		}
	}

	return retPoints, retType, nil
}

// userCheck checks user authentication
// returns nil, nil if user is not found
func (sdb *DbSqlite) userCheck(email, password string) (data.Nodes, error) {
	var ret []data.NodeEdge

	rows, err := sdb.db.Query("SELECT node_id FROM node_points WHERE type=? AND TEXT=?",
		data.PointTypeNodeType, data.NodeTypeUser)
	if err != nil {
		return nil, fmt.Errorf("userCheck, error query error: %v", err)
	}
	defer rows.Close()

	var ids []string

	for rows.Next() {
		var id string
		err = rows.Scan(&id)
		if err != nil {
			log.Println("Error scanning user id: ", id)
			continue
		}

		ids = append(ids, id)
	}

	if err := rows.Close(); err != nil {
		return ret, err
	}

	for _, id := range ids {
		ne, err := sdb.getNodes(nil, "all", id, "", false)
		if err != nil {
			log.Println("Error getting user node for id: ", id)
			continue
		}
		if len(ne) < 1 {
			continue
		}

		n := ne[0].ToNode()
		u := n.ToUser()
		if u.Email == email && u.Pass == password {
			ret = append(ret, ne...)
		}
	}

	return ret, nil
}

// up returns upstream ids for a node
func (sdb *DbSqlite) up(id string, includeDeleted bool) ([]string, error) {
	var edgeIDs []string
	var ups []string

	edges, err := sdb.edges(nil, "SELECT * FROM edges WHERE down=?", id)
	if err != nil {
		return nil, err
	}

	for _, e := range edges {
		ups = append(ups, e.Up)
		edgeIDs = append(edgeIDs, e.ID)
	}

	if includeDeleted {
		return ups, nil
	}

	var ret []string

	for i, edgeID := range edgeIDs {
		points, _, err := sdb.queryPoints(nil,
			"SELECT * FROM edge_points WHERE edge_id=?", edgeID)
		if err != nil {
			return nil, fmt.Errorf("up error getting edge points: %v", err)
		}

		p, _ := points.Find(data.PointTypeTombstone, "")
		if p.Value == 0 {
			ret = append(ret, ups[i])
		}
	}

	return ret, nil
}
