package store

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
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
	JWTKey  []byte `json:"jwtKey"`
}

// NewSqliteDb creates a new Sqlite data store
func NewSqliteDb(dbFile string, rootID string) (*DbSqlite, error) {
	ret := &DbSqlite{}

	pragmas := "_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(8000)&_pragma=journal_size_limit(100000000)"

	dbFileOptions := fmt.Sprintf("%s?%s", dbFile, pragmas)

	log.Println("Open store: ", dbFileOptions)

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
				root_id TEXT,
			  jwt_key BLOB)`)
	if err != nil {
		return nil, fmt.Errorf("Error creating meta table: %v", err)
	}

	// check if jwt_key column exists
	row := db.QueryRow(`SELECT COUNT(*) AS CNTREC FROM pragma_table_info('meta') WHERE name='jwt_key'`)
	var count int
	err = row.Scan(&count)
	if err != nil {
		return nil, err
	}

	if count <= 0 {
		_, err := db.Exec(`ALTER TABLE meta ADD COLUMN jwt_key BLOB`)
		if err != nil {
			return nil, err
		}
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS edges (id TEXT NOT NULL PRIMARY KEY,
				up TEXT,
				down TEXT,
				hash INT,
				type TEXT)`)

	if err != nil {
		return nil, fmt.Errorf("Error creating edges table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS node_points (id TEXT NOT NULL PRIMARY KEY,
				node_id TEXT,
				type TEXT,
				key TEXT,
				time INT,
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
				time INT,
				idx REAL,
				value REAL,
				text TEXT,
				data BLOB,
				tombstone INT,
				origin TEXT)`)

	if err != nil {
		return nil, fmt.Errorf("Error creating edge_points table: %v", err)
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS edgeUp ON edges(up)`)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS edgeDown ON edges(down)`)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS edgeType ON edges(type)`)
	if err != nil {
		return nil, err
	}

	err = ret.initMeta()
	if err != nil {
		return nil, fmt.Errorf("Error initializing db meta: %v", err)
	}

	err = ret.runMigrations()
	if err != nil {
		return nil, fmt.Errorf("Error running migrations: %v", err)
	}

	if ret.meta.RootID == "" {
		// we need to initialize root node and user
		ret.meta.RootID, err = ret.initRoot(rootID)
		if err != nil {
			return nil, fmt.Errorf("Error initializing root node: %v", err)
		}
	}

	if len(ret.meta.JWTKey) <= 0 {
		err := ret.initJwtKey()
		if err != nil {
			return nil, fmt.Errorf("Error initializing JWT Key: %v", err)
		}
	}

	// make sure we find root ID
	nodes, err := ret.getNodes(nil, "all", ret.meta.RootID, "", false)
	if err != nil {
		return nil, fmt.Errorf("error fetching root node: %v", err)
	}

	if len(nodes) < 1 {
		return nil, fmt.Errorf("root node not found")
	}

	return ret, nil
}

func (sdb *DbSqlite) initMeta() error {
	// should be one row in the meta database
	rows, err := sdb.db.Query("SELECT id, version, root_id, jwt_key FROM meta")
	if err != nil {
		return err
	}
	defer rows.Close()

	var count int

	for rows.Next() {
		count++
		err = rows.Scan(&sdb.meta.ID, &sdb.meta.Version, &sdb.meta.RootID, &sdb.meta.JWTKey)
		if err != nil {
			return fmt.Errorf("Error scanning meta row: %v", err)
		}
	}

	if count < 1 {
		_, err := sdb.db.Exec("INSERT INTO meta(id, version, root_id) VALUES(?, ?, ?)", 0, 0, "")
		if err != nil {
			return err
		}
	}

	return nil
}

type point struct {
	data.Point
	id     string
	nodeId string
}

func (sdb *DbSqlite) runMigrations() error {
	if sdb.meta.Version < 3 {
		log.Println("DB: running migration 3")

		_, err := sdb.db.Exec(`ALTER TABLE node_points RENAME TO node_points_old`)
		if err != nil {
			return fmt.Errorf("Error moving table node_points: %v", err)
		}

		_, err = sdb.db.Exec(`ALTER TABLE edge_points RENAME TO edge_points_old`)
		if err != nil {
			return fmt.Errorf("Error moving table edge_points: %v", err)
		}

		_, err = sdb.db.Exec(`CREATE TABLE node_points (id TEXT NOT NULL PRIMARY KEY,
				node_id TEXT,
				type TEXT,
				key TEXT,
				time INT,
				idx REAL,
				value REAL,
				text TEXT,
				data BLOB,
				tombstone INT,
				origin TEXT)`)

		if err != nil {
			return fmt.Errorf("Error creating node_points table: %v", err)
		}

		_, err = sdb.db.Exec(`CREATE TABLE edge_points (id TEXT NOT NULL PRIMARY KEY,
				edge_id TEXT,
				type TEXT,
				key TEXT,
				time INT,
				idx REAL,
				value REAL,
				text TEXT,
				data BLOB,
				tombstone INT,
				origin TEXT)`)

		if err != nil {
			return fmt.Errorf("Error creating node_points table: %v", err)
		}

		// copy old node_points data to new table
		rows, err := sdb.db.Query("SELECT * FROM node_points_old")
		if err != nil {
			return err
		}
		defer rows.Close()

		var dbPoints []point

		for rows.Next() {
			var p point
			var timeS, timeNS int64
			err := rows.Scan(&p.id, &p.nodeId, &p.Type, &p.Key, &timeS, &timeNS, &p.Index, &p.Value, &p.Text,
				&p.Data, &p.Tombstone, &p.Origin)
			if err != nil {
				return err
			}
			p.Time = time.Unix(timeS, timeNS)
			dbPoints = append(dbPoints, p)
		}

		for _, p := range dbPoints {
			_, err := sdb.db.Exec(`INSERT INTO node_points(id, node_id, type, key, time,
				idx, value, text, data, tombstone, origin) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				p.id, p.nodeId, p.Type, p.Key, p.Time.UnixNano(), p.Index, p.Value, p.Text, p.Data, p.Tombstone,
				p.Origin)
			if err != nil {
				return fmt.Errorf("Error writing to new node_points table: %v", err)
			}
		}

		if err := rows.Close(); err != nil {
			return fmt.Errorf("Error closing rows: %v", err)
		}

		// copy old edge_points data to new table
		rows, err = sdb.db.Query("SELECT * FROM edge_points_old")
		if err != nil {
			return err
		}
		defer rows.Close()

		dbPoints = []point{}

		for rows.Next() {
			var p point
			var timeS, timeNS int64
			err := rows.Scan(&p.id, &p.nodeId, &p.Type, &p.Key, &timeS, &timeNS, &p.Index, &p.Value, &p.Text,
				&p.Data, &p.Tombstone, &p.Origin)
			if err != nil {
				return err
			}
			p.Time = time.Unix(timeS, timeNS)
			dbPoints = append(dbPoints, p)
		}

		for _, p := range dbPoints {
			_, err := sdb.db.Exec(`INSERT INTO edge_points(id, edge_id, type, key, time,
				idx, value, text, data, tombstone, origin) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				p.id, p.nodeId, p.Type, p.Key, p.Time.UnixNano(), p.Index, p.Value, p.Text, p.Data, p.Tombstone,
				p.Origin)
			if err != nil {
				return fmt.Errorf("Error writing to new node_points table: %v", err)
			}
		}

		if err := rows.Close(); err != nil {
			return fmt.Errorf("Error closing rows: %v", err)
		}

		_, err = sdb.db.Exec("DROP TABLE node_points_old")
		if err != nil {
			return fmt.Errorf("Error dropping table: %v", err)
		}

		_, err = sdb.db.Exec("DROP TABLE edge_points_old")
		if err != nil {
			return fmt.Errorf("Error dropping table: %v", err)
		}

		_, err = sdb.db.Exec(`UPDATE meta SET version = 3`)
		if err != nil {
			return err
		}
		sdb.meta.Version = 3
	}

	return nil
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
	// preserve root ID
	sdb.meta.RootID, err = sdb.initRoot(sdb.meta.RootID)
	if err != nil {
		return fmt.Errorf("error initializing root node: %v", err)
	}

	// make sure we find root ID
	nodes, err := sdb.getNodes(nil, "all", sdb.meta.RootID, "", false)
	if err != nil {
		return fmt.Errorf("error fetching root node: %v", err)
	}

	if len(nodes) < 1 {
		return fmt.Errorf("root node not found")
	}

	return nil
}

// verifyNodeHashes recursively verifies all the hash values for all nodes
// this walks to the bottom of the tree, and then works its way back up
func (sdb *DbSqlite) verifyNodeHashes(fix bool) error {
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

		// it's important to go through children first as this can
		// impact the current hash
		for _, c := range children {
			err := verify(c)
			if err != nil {
				return err
			}
		}

		hash := node.CalcHash(children)

		if hash != node.Hash {
			log.Printf("Hash failed for %v, stored: %v, calc: %v",
				node.ID, node.Hash, hash)
			if fix {
				log.Println("fixing ...")
				_, err := tx.Exec(`UPDATE edges SET hash = ? WHERE up = ? AND down = ?`,
					hash, node.Parent, node.ID)
				if err != nil {
					return err
				}
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

func (sdb *DbSqlite) initRoot(rootID string) (string, error) {
	log.Println("STORE: Initialize root node and admin user")
	rootNode := data.NodeEdge{
		ID:   rootID,
		Type: data.NodeTypeDevice,
	}

	rootNode.ID = rootID

	if rootNode.ID == "" {
		rootNode.ID = uuid.New().String()
	}

	err := sdb.nodePoints(rootNode.ID, rootNode.Points)
	if err != nil {
		return "", fmt.Errorf("Error setting root node points: %v", err)
	}

	err = sdb.edgePoints(rootNode.ID, "root", data.Points{
		{Type: data.PointTypeTombstone, Value: 0},
		{Type: data.PointTypeNodeType, Text: rootNode.Type},
	})
	if err != nil {
		return "", fmt.Errorf("Error sending root node edges: %w", err)
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

	err = sdb.nodePoints(admin.ID, points)
	if err != nil {
		return "", fmt.Errorf("Error setting default user: %v", err)
	}

	err = sdb.edgePoints(admin.ID, rootNode.ID, data.Points{
		{Type: data.PointTypeTombstone, Value: 0},
		{Type: data.PointTypeNodeType, Text: data.NodeTypeUser},
	})

	if err != nil {
		return "", err
	}

	sdb.writeLock.Lock()
	defer sdb.writeLock.Unlock()
	_, err = sdb.db.Exec("UPDATE meta SET root_id = ?", rootNode.ID)
	if err != nil {
		return "", fmt.Errorf("Error setting meta rootID: %v", err)
	}

	return rootNode.ID, nil
}

func (sdb *DbSqlite) initJwtKey() error {
	var f *os.File
	f, err := os.Open("/dev/urandom")
	if err != nil {
		return fmt.Errorf("Error opening /dev/urandom: %v", err)
	}
	defer f.Close()
	sdb.meta.JWTKey = make([]byte, 20)
	_, err = f.Read(sdb.meta.JWTKey)

	if err != nil {
		return fmt.Errorf("Error reading urandom to make key: %v", err)
	}

	sdb.writeLock.Lock()
	defer sdb.writeLock.Unlock()
	_, err = sdb.db.Exec("UPDATE meta SET jwt_key = ?", sdb.meta.JWTKey)
	if err != nil {
		return fmt.Errorf("Error setting meta jwt key: %v", err)
	}

	return nil
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
		var timeNS int64
		var pID string
		var nodeID string
		err := rowsPoints.Scan(&pID, &nodeID, &p.Type, &p.Key, &timeNS, &p.Index, &p.Value, &p.Text,
			&p.Data, &p.Tombstone, &p.Origin)
		if err != nil {
			rollback()
			return err
		}
		p.Time = time.Unix(0, timeNS)
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

	stmt, err := tx.Prepare(`INSERT INTO node_points(id, node_id, type, key, time,
                 idx, value, text, data, tombstone, origin)
		 VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		 type = ?3,
		 key = ?4,
		 time = ?5,
		 idx = ?6,
		 value = ?7,
		 text = ?8,
		 data = ?9,
		 tombstone = ?10,
		 origin = ?11
		 `)
	defer stmt.Close()

	for i, p := range writePoints {
		tNs := p.Time.UnixNano()
		pID := writePointIDs[i]
		_, err = stmt.Exec(pID, id, p.Type, p.Key, tNs, p.Index, p.Value, p.Text, p.Data, p.Tombstone,
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
	if nodeID == parentID {
		return fmt.Errorf("Error: edgePoints nodeID=parentID=%v", nodeID)
	}

	if nodeID == sdb.meta.RootID {
		for _, p := range points {
			if p.Type == data.PointTypeTombstone && p.Value > 0 {
				return fmt.Errorf("Error, can't delete root node")
			}
		}
	}

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
		var timeNS int64
		var pID string
		var nodeID string
		err := rowsPoints.Scan(&pID, &nodeID, &p.Type, &p.Key, &timeNS, &p.Index, &p.Value, &p.Text,
			&p.Data, &p.Tombstone, &p.Origin)
		if err != nil {
			rollback()
			return err
		}
		p.Time = time.Unix(0, timeNS)
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

	var nodeType string

NextPin:
	for _, pIn := range points {
		// we don't store node type points
		if pIn.Type == data.PointTypeNodeType {
			nodeType = pIn.Text
			continue NextPin
		}

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
	stmt, err := tx.Prepare(`INSERT INTO edge_points(id, edge_id, type, key, time,
                 idx, value, text, data, tombstone, origin)
		 VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		 type = ?3,
		 key = ?4,
		 time = ?5,
		 idx = ?6,
		 value = ?7,
		 text = ?8,
		 data = ?9,
		 tombstone = ?10,
		 origin = ?11
		 `)

	for i, p := range writePoints {
		tNs := p.Time.UnixNano()
		pID := writePointIDs[i]
		_, err = stmt.Exec(pID, edge.ID, p.Type, p.Key, tNs, p.Index, p.Value, p.Text, p.Data, p.Tombstone,
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
		if nodeType == "" {
			rollback()
			return fmt.Errorf("Node type must be sent with new edges")
		}
		// did not find edge, need to add it
		edge.Up = parentID
		edge.Down = nodeID
		edge.Type = nodeType

		// look for existing node points that must be added to the hash
		rowsPoints, err := tx.Query("SELECT * FROM node_points WHERE node_id=?", nodeID)
		if err != nil {
			rollback()
			return err
		}
		defer rowsPoints.Close()

		for rowsPoints.Next() {
			var p data.Point
			var timeNS int64
			var pID string
			var nodeID string
			err := rowsPoints.Scan(&pID, &nodeID, &p.Type, &p.Key, &timeNS, &p.Index, &p.Value, &p.Text,
				&p.Data, &p.Tombstone, &p.Origin)
			if err != nil {
				rollback()
				return err
			}
			p.Time = time.Unix(0, timeNS)
			hashUpdate ^= p.CRC()
		}

		if err := rowsPoints.Close(); err != nil {
			rollback()
			return fmt.Errorf("Error closing rowsPoints: %v", err)
		}

		_, err = tx.Exec(`INSERT INTO edges(id, up, down, hash, type) VALUES (?, ?, ?, ?, ?)`,
			edge.ID, edge.Up, edge.Down, 0, edge.Type)

		// (hash will be populated later)

		if err != nil {
			log.Println("edge insert failed, trying again ...: ", err)
			// FIXME, occasionally the above INSERT will fail with "database is locked (5) (SQLITE_BUSY)"
			// FIXME, not sure if retry is required any more since we removed the nested
			// queries
			// not sure why, but the below retry seems to work around this issue for now
			_, err := tx.Exec(`INSERT INTO edges(id, up, down, hash, type) VALUES (?, ?, ?, ?)`,
				edge.ID, edge.Up, edge.Down, edge.Hash, edge.Type)

			// TODO check for downstream node and add in its hash
			if err != nil {
				rollback()
				return fmt.Errorf("Error when writing edge: %v", err)
			}
		}
	}

	err = sdb.updateHash(tx, nodeID, hashUpdate)
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
		err = rowsEdges.Scan(&edge.ID, &edge.Up, &edge.Down, &edge.Hash, &edge.Type)
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

// If parent is set to "all", then all instances of the node are returned.
// If parent is set and id is "all", then all child nodes are returned.
// Parent can be set to "root" and id to "all" to fetch the root node(s).
func (sdb *DbSqlite) getNodes(tx *sql.Tx, parent, id, typ string, includeDel bool) ([]data.NodeEdge, error) {
	var ret []data.NodeEdge

	if parent == "" || parent == "none" {
		return nil, errors.New("Parent must be set to valid ID, or all")
	}

	if id == "" {
		id = "all"
	}

	var q string

	switch {
	case parent == "all" && id == "all":
		return nil, errors.New("invalid combination of parent and id")
	case parent == "all":
		q = fmt.Sprintf("SELECT * FROM edges WHERE down = '%v'", id)
	case id == "all":
		q = fmt.Sprintf("SELECT * FROM edges WHERE up = '%v'", parent)
	default:
		// both parent and id are specified
		q = fmt.Sprintf("SELECT * FROM edges WHERE up='%v' AND down = '%v'", parent, id)
	}

	if typ != "" {
		q += fmt.Sprintf("AND type = '%v'", typ)
	}

	edges, err := sdb.edges(tx, q)

	if err != nil {
		return ret, err
	}

	if len(edges) < 1 {
		return ret, nil
	}

	for _, edge := range edges {
		var ne data.NodeEdge
		ne.ID = edge.Down
		ne.Parent = edge.Up
		ne.Hash = edge.Hash
		ne.Type = edge.Type

		ne.EdgePoints, err = sdb.queryPoints(tx,
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

		ne.Points, err = sdb.queryPoints(tx,
			"SELECT * FROM node_points WHERE node_id=?", edge.Down)
		if err != nil {
			return nil, fmt.Errorf("children error getting node points: %v", err)
		}

		ret = append(ret, ne)
	}

	return ret, nil
}

// returns points, and error
func (sdb *DbSqlite) queryPoints(tx *sql.Tx, query string, args ...any) (data.Points, error) {
	var retPoints data.Points
	rowsPoints, err := sdb.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rowsPoints.Close()

	for rowsPoints.Next() {
		var p data.Point
		var timeNS int64
		var pID string
		var nodeID string
		err := rowsPoints.Scan(&pID, &nodeID, &p.Type, &p.Key, &timeNS, &p.Index, &p.Value, &p.Text,
			&p.Data, &p.Tombstone, &p.Origin)
		if err != nil {
			return nil, err
		}
		p.Time = time.Unix(0, timeNS)
		retPoints = append(retPoints, p)
	}

	return retPoints, nil
}

// userCheck checks user authentication
// returns nil, nil if user is not found
func (sdb *DbSqlite) userCheck(email, password string) (data.Nodes, error) {
	var ret []data.NodeEdge

	rows, err := sdb.db.Query("SELECT down FROM edges WHERE type=?", data.NodeTypeUser)
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
		points, err := sdb.queryPoints(nil,
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
