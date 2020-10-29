package main

import (
	"fmt"
	"log"

	"github.com/dgraph-io/badger/v2"
	"github.com/genjidb/genji"
	"github.com/genjidb/genji/engine/badgerengine"
)

type value struct {
	Name string
}

// returns true if db already exists
func setup(db *genji.DB) {
}

func main() {
	//db, err := genji.Open(":memory:")
	//db, err := genji.Open("genji-test6.db")

	// Create a badger engine
	ng, err := badgerengine.NewEngine(badger.DefaultOptions("mydb"))
	if err != nil {
		log.Fatal(err)
	}

	// Pass it to genji
	db, err := genji.New(ng)

	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	err = db.Exec("CREATE TABLE IF NOT EXISTS t1;")
	if err != nil {
		log.Fatal("error creating t1: ", err)
	}

	err = db.Exec("CREATE TABLE IF NOT EXISTS t2;")
	if err != nil {
		log.Fatal("error creating t2: ", err)
	}

	// insert a bunch of records into db
	for i := 0; i < 1000; i++ {
		err = db.Exec(`insert into t1 values {name: 't1'}`)
		if err != nil {
			log.Fatal("error inserting into t1: ", err)
		}

		err = db.Exec(`insert into t2 values {name: 't2'}`)
		if err != nil {
			log.Fatal("error inserting into t1: ", err)
		}
	}

	go func() {
		t1Count := 0
		for {
			res, err := db.Query(`select * from t1`)
			if err != nil {
				log.Fatal("error query t1: ", err)
			}
			res.Close()
			t1Count++
			fmt.Println("t1Count: ", t1Count)
			//time.Sleep(10 * time.Millisecond)
		}
	}()

	t2Count := 0
	for {
		res, err := db.Query(`select * from t2`)
		if err != nil {
			log.Fatal("error query t2: ", err)
		}
		res.Close()
		t2Count++
		fmt.Println("t2Count: ", t2Count)
		//time.Sleep(11 * time.Millisecond)
	}
}
