package main

import (
	"log"

	"github.com/genjidb/genji"
)

// User type
type User struct {
	ID     string
	Name   string
	Groups []string
}

// returns true if db already exists
func setup(db *genji.DB) {
	err := db.Exec("CREATE TABLE IF NOT EXISTS users;")
	if err != nil {
		log.Fatal("error creating users: ", err)
	}

	err = db.Exec("CREATE INDEX IF NOT EXISTS idx_user_groups ON users(groups)")
	if err != nil {
		log.Fatal("error creating index: ", err)
	}
}

func populateData(db *genji.DB) {
	// insert first user, then a lot of another user
	u1 := User{
		Name:   "fred",
		Groups: []string{"g1", "g2", "g3"},
	}

	count := 100
	for i := 0; i < count; i++ {
		err := db.Exec("INSERT INTO users VALUES ?", &u1)
		if err != nil {
			log.Fatal("Error inserting user: ", err)
		}
	}

	u2 := User{
		Name:   "mary",
		Groups: []string{"g4", "g5", "g6"},
	}

	err := db.Exec("INSERT INTO users VALUES ?", &u2)
	if err != nil {
		log.Fatal("Error inserting user: ", err)
	}

}

/*
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
*/

func main() {
	//db, err := genji.Open(":memory:")
	db, err := genji.Open("genji-test3.db")

	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	setup(db)
	populateData(db)

	/*
		// look for the record at the beginning of the collection
		count1 := query(db, `SELECT * FROM tests WHERE field1 = "hi"`)
		count2 := query(db, `SELECT * FROM tests WHERE field2 = "there"`)

		if count1 != count2 {
			log.Printf("indexed field returned %v records, non indexed filed returned %v records, expected 100 records for both", count1, count2)
		}

		log.Println("All done :-)")
	*/
}
