package isdb

import (
	"path"

	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/timshannon/bolthold"
)

// IsDb is used for all db access in the application.
// We will eventually turn this into an interface to
// handle multiple Db backends.
type IsDb struct {
	store *bolthold.Store
}

// NewDb creates a new Db instance for the app
func NewDb(dataDir string) (*IsDb, error) {
	dbFile := path.Join(dataDir, "data.db")
	store, err := bolthold.Open(dbFile, 0666, nil)
	if err != nil {
		return nil, err
	}

	return &IsDb{
		store: store,
	}, nil
}

// ReadConfig reads the IS config from the database
func (db *IsDb) ReadConfig(config *isdata.Config) error {
	err := db.store.Get(0, config)

	if err != nil {
		if err == bolthold.ErrNotFound {
			// data is not stored, so simply return zero'd config
			return nil
		}

		// there was an error reading so return error
		return err
	}

	return nil
}

// WriteConfig writes the IS config to the database
func (db *IsDb) WriteConfig(config *isdata.Config) error {

	return db.store.Upsert(0, config)
}
