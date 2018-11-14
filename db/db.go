package db

import "github.com/timshannon/bolthold"

// Open the db store and return handle
func Open() (*bolthold.Store, error) {
	return bolthold.Open("data.db", 0666, nil)
}
