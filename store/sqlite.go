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

// NewSqliteDb creates a new Sqlite data store
func NewSqliteDb(dataDir string, dbFile string) (*DbSqlite, error) {
	ret := &DbSqlite{}

	db, err := sql.Open("sqlite", dbFile)
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

	return ret, nil
}

func (sdb *DbSqlite) initRoot() (string, error) {
	log.Println("NODE: Initialize root node and admin user")
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
		err := rowsPoints.Scan(&pID, nil, &p.Type, &p.Key, &timeS, &timeNS, &p.Index, &p.Value, &p.Text,
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
		 type = ?2,
		 key = ?3,
		 time_s = ?4,
		 time_ns = ?5,
		 idx = ?6,
		 value = ?7,
		 text = ?8,
		 data = ?9,
		 tombstone = ?10,
		 origin = ?11
		 `)
	defer stmt.Close()

	for i, p := range writePoints {
		tS := p.Time.Unix()
		tNs := p.Time.UnixNano() - 1e9*tS
		pID := writePointIDs[i]
		_, err = stmt.Exec(pID, id, p.Type, p.Key, tS, tNs, p.Index, p.Value, p.Text, p.Data, p.Tombstone,
			p.Origin)
		if err != nil {
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
		err := rowsPoints.Scan(&pID, nil, &p.Type, &p.Key, &timeS, &timeNS, &p.Index, &p.Value, &p.Text,
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
		 type = ?2,
		 key = ?3,
		 time_s = ?4,
		 time_ns = ?5,
		 idx = ?6,
		 value = ?7,
		 text = ?8,
		 data = ?9,
		 tombstone = ?10,
		 origin = ?11
		 `)
	defer stmt.Close()

	for i, p := range writePoints {
		tS := p.Time.Unix()
		tNs := p.Time.UnixNano() - 1e9*tS
		pID := writePointIDs[i]
		_, err = stmt.Exec(pID, edge.ID, p.Type, p.Key, tS, tNs, p.Index, p.Value, p.Text, p.Data, p.Tombstone,
			p.Origin)
		if err != nil {
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
	return nil, errors.New("not implemented")
}
