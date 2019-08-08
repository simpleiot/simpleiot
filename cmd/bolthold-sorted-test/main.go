package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/timshannon/bolthold"
)

type testData struct {
	Value int
	Time  time.Time
}

// db is used for a test database
type db struct {
	filename string
	store    *bolthold.Store
}

// newDb creates a new db instance for the test
func newDb(dataDir string) (*db, error) {
	dbFile := path.Join(dataDir, "data.db")
	store, err := bolthold.Open(dbFile, 0666, nil)
	if err != nil {
		return nil, err
	}

	return &db{
		filename: dbFile,
		store:    store,
	}, nil
}

// readTest returns a slice holding all test data from database
func (db *db) readTest() ([]testData, error) {
	var samples []testData
	err := db.store.Find(&samples, nil)

	if err != nil {
		if err == bolthold.ErrNotFound {
			// data is not stored, so simply return nil array and nil error
			return nil, nil
		}

		// there was an error reading so return nil array and error
		return nil, err
	}

	return samples, nil
}

func main() {
	testData1 := testData{
		Value: 100,
		Time:  time.Now(),
	}
	testData2 := testData{
		Value: 200,
		Time:  time.Now(),
	}
	testData3 := testData{
		Value: 300,
		Time:  time.Now(),
	}

	err := os.RemoveAll("./temp")

	if err != nil {
		fmt.Println("failed to remove file: ", err)
	}

	os.Mkdir("./temp", os.ModePerm)

	db, err := newDb("./temp")

	if err != nil {
		log.Fatal("failed to open db: ", err)
	}

	writeHandleErr(db, &testData1, time.Now().Add(-2*time.Hour))
	writeHandleErr(db, &testData2, time.Now())
	writeHandleErr(db, &testData3, time.Now().Add(4*time.Minute))

	testR, err := db.readTest()

	if err != nil {
		fmt.Println("failed reading data: ", err)
	}

	for _, test := range testR {
		fmt.Println(test.Value)
	}
}

func writeHandleErr(db *db, testData *testData, time time.Time) {
	err := db.store.Insert(time, testData)

	if err != nil {
		fmt.Println("failed writing data: ", err)
	}
}
