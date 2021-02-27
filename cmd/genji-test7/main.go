package main

import (
	"log"
	"os/exec"

	"github.com/genjidb/genji"
	"github.com/genjidb/genji/database"
)

// Test data value
type Test struct {
	ID    string
	Value string
}

func main() {
	//db, err := genji.Open(":memory:")

	exec.Command("rm", "genji-test7.db").Run()

	db, err := genji.Open("genji-test7.db")

	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	err = db.Exec("CREATE TABLE test (id TEXT PRIMARY KEY);")
	if err != nil {
		if err != database.ErrTableAlreadyExists {
			log.Fatal("error creating tests: ", err)
		}
	}

	err = db.Exec(`INSERT INTO test VALUES ?`, Test{ID: "abc", Value: "hi there"})
	if err != nil {
		log.Fatal("error inserting: ", err)
	}

	err = db.Exec(`UPDATE test SET value=? where id=?`, "bye there", "abc")
	if err != nil {
		log.Fatal("error updating: ", err)
	}

	log.Println("All done :-)")
}
