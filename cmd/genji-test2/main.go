package main

import (
	"log"
	"time"

	"github.com/genjidb/genji"
	"github.com/genjidb/genji/database"
	"github.com/genjidb/genji/document"
)

// Test type
type Test struct {
	Field1 string
	Field2 string
}

// returns true if db already exists
func setup(db *genji.DB) bool {
	err := db.Exec("CREATE TABLE tests;")
	if err != nil {
		if err != database.ErrTableAlreadyExists {
			log.Fatal("error creating tests: ", err)
		}
	}

	err = db.Exec("CREATE INDEX idx_tests_field1 ON tests(field1)")
	if err != nil {
		if err != database.ErrIndexAlreadyExists {
			log.Fatal("error creating index: ", err)
		}
		return true
	}

	return false
}

func populateData(db *genji.DB) {
	// insert first user, then a lot of another user
	t := Test{
		Field1: "hi",
		Field2: "there",
	}

	count := 100
	for i := 0; i < count; i++ {
		err := db.Exec("INSERT INTO tests VALUES ?", &t)
		if err != nil {
			log.Fatal("Error inserting test: ", err)
		}
	}
}

// returns # of docs found
func query(db *genji.DB, q string) int {
	start := time.Now()

	res, err := db.Query(q)

	if err != nil {
		log.Fatal("query error: ", err)
	}

	defer res.Close()

	count := 0

	err = res.Iterate(func(d document.Document) error {
		t := Test{}
		err := document.StructScan(d, &t)
		if err != nil {
			log.Fatal("Error scanning document: ", err)
		}

		count++

		return nil
	})

	log.Printf("%v: documents found: %v, time: %v", q, count, time.Since(start))
	return count
}

func main() {
	//db, err := genji.Open(":memory:")
	db, err := genji.Open("genji-test2.db")

	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	dataExists := setup(db)

	if !dataExists {
		populateData(db)
	}

	// look for the record at the beginning of the collection
	count1 := query(db, `SELECT * FROM tests WHERE field1 = "hi"`)
	count2 := query(db, `SELECT * FROM tests WHERE field2 = "there"`)

	if count1 != count2 {
		log.Printf("indexed field returned %v records, non indexed filed returned %v records, expected 100 records for both", count1, count2)
	}

	log.Println("All done :-)")
}
