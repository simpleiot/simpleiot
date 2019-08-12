package main

import (
	"fmt"
	"log"
	"time"

	"github.com/simpleiot/simpleiot/data"
	"github.com/timshannon/bolthold"
)

func main() {
	db, err := bolthold.Open("temp.db", 0666, nil)

	if err != nil {
		log.Fatal("Could not open db: ", err)
	}

	defer db.Close()

	s := data.Sample{
		Type:  "test",
		Value: 12,
	}

	err = db.Insert(time.Now(), s)
	if err != nil {
		log.Fatal("Error inserting: ", err)
	}

	s.Type = "test2"

	err = db.Insert(time.Now(), s)
	if err != nil {
		log.Fatal("Error inserting: ", err)
	}

	read := []data.Sample{}

	err = db.Find(&read, nil)

	fmt.Printf("data read: %+v\n", read)

}
