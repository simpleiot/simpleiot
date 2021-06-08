package main

import (
	"fmt"
	"log"

	"github.com/genjidb/genji"
	"github.com/genjidb/genji/document"
)

// Edge is used to describe the relationship
// between two nodes
type Edge struct {
	ID        string `json:"id"`
	Up        string `json:"up"`
	Down      string `json:"down"`
	Tombstone bool   `json:"tombstone"`
}

func main() {
	db, err := genji.Open("test.db")
	if err != nil {
		log.Fatal("Error opening db: ", err)
	}

	err = db.Exec(`CREATE TABLE IF NOT EXISTS edges (id TEXT PRIMARY KEY)`)
	if err != nil {
		log.Fatal("Error creating edges table: ", err)
	}

	err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_edge_up ON edges(up)`)
	if err != nil {
		log.Fatal("Error creating idx_edge_up: ", err)
	}

	err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_edge_down ON edges(down)`)
	if err != nil {
		log.Fatal("Error creating idx_edge_down: ", err)
	}

	// nodes
	// 1 root
	// 2 admin
	// 3 group 1
	// 4 user 1

	err = db.Exec(`insert into edges values ?`, Edge{"10", "1", "2", false})
	if err != nil {
		log.Fatal("1: ", err)
	}

	err = db.Exec(`insert into edges values ?`, Edge{"11", "1", "3", false})
	if err != nil {
		log.Fatal("2: ", err)
	}

	err = db.Exec(`insert into edges values ?`, Edge{"12", "3", "4", false})
	if err != nil {
		log.Fatal("3: ", err)
	}

	err = db.Exec(`insert into edges values ?`, Edge{"13", "1", "4", false})
	if err != nil {
		log.Fatal("4: ", err)
	}

	err = db.Exec(`update edges set tombstone = true where down = ? and up = ?`,
		"4", "3")
	if err != nil {
		log.Fatal("5: ", err)
	}

	err = db.Exec(`update edges set tombstone = true where down = ? and up = ?`,
		"4", "1")
	if err != nil {
		log.Fatal("6: ", err)
	}

	res, _ := db.Query(`select * from edges where up = ?`, "3")
	res.Iterate(func(d document.Document) error {
		var edge Edge
		err = document.StructScan(d, &edge)
		if err != nil {
			return err
		}

		fmt.Printf("Edge: %+v\n", edge)

		return nil
	})
}
