package main

import (
	"log"
	"os/exec"

	"github.com/genjidb/genji"
)

// Node represents the state of a device. UUID is recommended
// for ID to prevent collisions is distributed instances.
type Node struct {
	ID          string
	Type        string
	Description string
}

func main() {
	// both memory and bolt show this issue
	//db, err := genji.Open(":memory:")

	exec.Command("rm", "genji-test9.db").Run()

	db, err := genji.Open("genji-test9.db")

	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	err = db.Exec(`CREATE TABLE IF NOT EXISTS nodes (id TEXT PRIMARY KEY)`)
	if err != nil {
		log.Fatal("Error creating nodes table: ", err)
	}

	// if this index is removed, this program runs fine, otherwise it fails with
	// Error updating node description2: key not found
	err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_nodes_type ON nodes(type)`)
	if err != nil {
		log.Fatal("Error creating idx_nodes_type: ", err)
	}

	rootNode := Node{Type: "device", ID: "1234"}

	err = db.Exec(`insert into nodes values ?`, rootNode)
	if err != nil {
		log.Fatal("Error creating root node: ", err)
	}

	err = db.Exec(`update nodes set description = ? where id = ?`,
		"hi", rootNode.ID)

	if err != nil {
		log.Fatal("Error updating node description1: ", err)
	}

	err = db.Exec(`update nodes set description = ? where id = ?`,
		"hi", rootNode.ID)

	if err != nil {
		log.Fatal("Error updating node description2: ", err)
	}

	err = db.Exec(`update nodes set description = ? where id = ?`,
		"hi", rootNode.ID)

	if err != nil {
		log.Fatal("Error updating node description3: ", err)
	}

	err = db.Exec(`update nodes set description = ? where id = ?`,
		"hi", rootNode.ID)

	if err != nil {
		log.Fatal("Error updating node description4: ", err)
	}

	log.Println("All done :-)")
}
