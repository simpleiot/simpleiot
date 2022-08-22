package store

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/simpleiot/simpleiot/data"

	// tell sql to use sqlite
	_ "modernc.org/sqlite"
)

// DbSqlite represents a SQLite data store
type DbSqlite struct {
	db   *sql.DB
	meta Meta
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
				hash TEXT)`)

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

	err := sdb.nodePoints(rootNode.ID, rootNode.Points)
	if err != nil {
		return "", fmt.Errorf("Error setting root node points: %v", err)
	}

	err = sdb.edgePoints(rootNode.ID, "", data.Points{{Type: data.PointTypeTombstone, Value: 0}})
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

	err = sdb.edgePoints(admin.ID, rootNode.ID, data.Points{{Type: data.PointTypeTombstone, Value: 0}})
	if err != nil {
		return "", err
	}

	_, err = sdb.db.Exec("INSERT INTO meta(id, version, root_id) VALUES(?, ?, ?)", 0, 0, rootNode.ID)
	if err != nil {
		return "", fmt.Errorf("Error setting meta data: %v", err)
	}

	return rootNode.ID, nil
}

func (sdb *DbSqlite) nodePoints(id string, points data.Points) error {
	rowsPoints, err := sdb.db.Query("SELECT * FROM node_points WHERE node_id=?", id)
	if err != nil {
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
			return err
		}
		p.Time = time.Unix(timeS, timeNS)
		dbPoints = append(dbPoints, p)
		dbPointIDs = append(dbPointIDs, pID)
	}

	var writePoints data.Points
	var writePointIDs []string

NextPin:
	for _, pIn := range points {
		if pIn.Time.IsZero() {
			pIn.Time = time.Now()
		}

		for j, pDb := range dbPoints {
			if pIn.Type == pDb.Type && pIn.Key == pDb.Key {
				// found a match
				if pIn.Time.After(pDb.Time) {
					writePoints = append(writePoints, pIn)
					writePointIDs = append(writePointIDs, dbPointIDs[j])
				}
				break NextPin
			}
		}

		// point was not found so write it
		writePoints = append(writePoints, pIn)
		writePointIDs = append(writePointIDs, uuid.New().String())
	}

	// loop through write points and write them
	tx, err := sdb.db.Begin()
	if err != nil {
		return err
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
			rbErr := tx.Rollback()
			if rbErr != nil {
				log.Println("Rollback error: ", rbErr)
			}
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (sdb *DbSqlite) edgePoints(nodeID, parentID string, points data.Points) error {
	if parentID == "" {
		parentID = "none"
	}

	rowsEdge, err := sdb.db.Query("SELECT * FROM edges WHERE up=? AND down=?", parentID, nodeID)
	if err != nil {
		return err
	}
	defer rowsEdge.Close()

	var edge data.Edge

	for rowsEdge.Next() {
		err := rowsEdge.Scan(&edge.ID, &edge.Up, &edge.Down, &edge.Hash)
		if err != nil {
			return err
		}
	}

	if edge.ID == "" {
		edge.ID = uuid.New().String()
		edge.Up = parentID
		edge.Down = nodeID

		// did not find edge, need to add it
		_, err := sdb.db.Exec(`INSERT INTO edges(id, up, down, hash) VALUES (?, ?, ?, ?)`,
			edge.ID, edge.Up, edge.Down, "")

		if err != nil {
			return err
		}
	}

	rowsPoints, err := sdb.db.Query("SELECT * FROM edge_points WHERE edge_id=?", edge.ID)
	if err != nil {
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
			return err
		}
		p.Time = time.Unix(timeS, timeNS)
		dbPoints = append(dbPoints, p)
		dbPointIDs = append(dbPointIDs, pID)
	}

	var writePoints data.Points
	var writePointIDs []string

NextPin:
	for _, pIn := range points {
		if pIn.Time.IsZero() {
			pIn.Time = time.Now()
		}

		for j, pDb := range dbPoints {
			if pIn.Type == pDb.Type && pIn.Key == pDb.Key {
				// found a match
				if pIn.Time.After(pDb.Time) {
					writePoints = append(writePoints, pIn)
					writePointIDs = append(writePointIDs, dbPointIDs[j])
				}
				break NextPin
			}
		}

		// point was not found so write it
		writePoints = append(writePoints, pIn)
		writePointIDs = append(writePointIDs, uuid.New().String())
	}

	// loop through write points and write them
	tx, err := sdb.db.Begin()
	if err != nil {
		return err
	}
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
	defer stmt.Close()

	for i, p := range writePoints {
		tS := p.Time.Unix()
		tNs := p.Time.UnixNano() - 1e9*tS
		pID := writePointIDs[i]
		_, err = stmt.Exec(pID, edge.ID, p.Type, p.Key, tS, tNs, p.Index, p.Value, p.Text, p.Data, p.Tombstone,
			p.Origin)
		if err != nil {
			rbErr := tx.Rollback()
			if rbErr != nil {
				log.Println("Rollback error: ", rbErr)
			}

			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

// Close the db
func (sdb *DbSqlite) Close() error {
	return sdb.db.Close()
}

func (sdb *DbSqlite) rootNodeID() string {
	return sdb.meta.RootID
}

func (sdb *DbSqlite) node(id string) (*data.Node, error) {
	var err error
	var ret data.Node
	ret.ID = id

	query := fmt.Sprintf("SELECT * FROM node_points WHERE node_id='%v'", id)
	ret.Points, ret.Type, err = sdb.queryPoints(query)

	if err != nil {
		return nil, err
	}

	if ret.Type == "" {
		return nil, errors.New("node not found")
	}

	return &ret, err
}

func (sdb *DbSqlite) children(id, typ string, includeDel bool) ([]data.NodeEdge, error) {
	var ret []data.NodeEdge

	rowsEdges, err := sdb.db.Query("SELECT * FROM edges WHERE up=?", id)
	if err != nil {
		return ret, fmt.Errorf("Error getting edges: %v", err)
	}
	defer rowsEdges.Close()

	for rowsEdges.Next() {
		var edge data.Edge
		err = rowsEdges.Scan(&edge.ID, &edge.Up, &edge.Down, &edge.Hash)
		if err != nil {
			return nil, fmt.Errorf("Error scanning edges: %v", err)
		}

		var ne data.NodeEdge
		ne.ID = edge.Down
		ne.Parent = id
		ne.Hash = edge.Hash

		q := fmt.Sprintf("SELECT * FROM edge_points WHERE edge_id='%v'", edge.ID)
		ne.EdgePoints, _, err = sdb.queryPoints(q)
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

		q = fmt.Sprintf("SELECT * FROM node_points WHERE node_id='%v'", edge.Down)
		ne.Points, ne.Type, err = sdb.queryPoints(q)
		if err != nil {
			return nil, fmt.Errorf("children error getting edge points: %v", err)
		}

		if typ != "" {
			if ne.Type != typ {
				// skip node of incorrect type
				continue
			}
		}

		ret = append(ret, ne)
	}

	return ret, nil
}

// id must be a valid ID or "root"
// parent can be:
//   - id of node
//   - none: parent details are skipped
//   - all: instances of node are fetched
func (sdb *DbSqlite) nodeEdge(id, parent string) ([]data.NodeEdge, error) {
	var ret []data.NodeEdge

	if id == "root" {
		id = sdb.meta.RootID
	}

	if parent == "" {
		parent = "none"
	}

	var q string

	switch parent {
	case "none":
		node, err := sdb.node(id)
		if err != nil {
			return ret, err
		}
		ne := node.ToNodeEdge(data.Edge{})
		return []data.NodeEdge{ne}, nil
	case "all":
		q = fmt.Sprintf("SELECT * FROM edges WHERE down = '%v'", id)
	default:
		q = fmt.Sprintf("SELECT * FROM edges WHERE up='%v' AND down = '%v'", parent, id)
	}

	rowsEdges, err := sdb.db.Query(q)
	if err != nil {
		return ret, fmt.Errorf("Error getting edges: %v", err)
	}
	defer rowsEdges.Close()

	for rowsEdges.Next() {
		var edge data.Edge
		err = rowsEdges.Scan(&edge.ID, &edge.Up, &edge.Down, &edge.Hash)
		if err != nil {
			return nil, fmt.Errorf("Error scanning edges: %v", err)
		}

		var ne data.NodeEdge
		ne.ID = edge.Down
		ne.Parent = edge.Up
		ne.Hash = edge.Hash

		q := fmt.Sprintf("SELECT * FROM edge_points WHERE edge_id='%v'", edge.ID)
		ne.EdgePoints, _, err = sdb.queryPoints(q)
		if err != nil {
			return nil, fmt.Errorf("children error getting edge points: %v", err)
		}

		q = fmt.Sprintf("SELECT * FROM node_points WHERE node_id='%v'", edge.Down)
		ne.Points, ne.Type, err = sdb.queryPoints(q)
		if err != nil {
			return nil, fmt.Errorf("children error getting node points: %v", err)
		}

		ret = append(ret, ne)
	}

	if len(ret) < 1 {
		return ret, fmt.Errorf("Node not found")
	}

	return ret, nil
}

func (sdb *DbSqlite) queryPoints(query string) (data.Points, string, error) {
	var retPoints data.Points
	var retType string
	rowsPoints, err := sdb.db.Query(query)
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

	for rows.Next() {
		var id string
		err = rows.Scan(&id)
		if err != nil {
			log.Println("Error scanning user id: ", id)
			continue
		}
		ne, err := sdb.nodeEdge(id, "all")
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
