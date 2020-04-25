package db

import (
	"fmt"

	"go.etcd.io/bbolt"
)

// BBoltCheck checks a database for errors.
// Because bbolt is a memory mapped database,
// the program may crash if the database is corrupted.
// Thus, this function should be called before
// the main app starts to verify the database is
// valid and any recovery action taken outside the
// application using bbolt.
func BBoltCheck(fileName string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("db crashed: %v", r)
		}
	}()

	db, err := bbolt.Open(fileName, 0666, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	return db.View(func(tx *bbolt.Tx) error {
		c := tx.Check()
		for {
			err, ok := <-c
			if !ok {
				return nil
			}

			if err != nil {
				return err
			}
		}
	})
}
