package app

import (
	"log"
	"path"

	"github.com/pkg/errors"
	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/farmation/isdb"
	"github.com/timshannon/bolthold"
)

// DbInit opens databases and initializes config and state
// returns config, state, and data databases
func DbInit(dataDir string, config *isdata.Config, state *isdata.State) (*isdb.IsDb, *isdb.IsDb, *isdb.IsDb, error) {
	dbFn := path.Join(dataDir, "data.db")
	dbConfigFn := path.Join(dataDir, "config.db")
	dbStateFn := path.Join(dataDir, "state.db")

	dbData, err := isdb.NewDb(dbFn)

	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "error opening data db")
	}

	dbConfig, err := isdb.NewDb(dbConfigFn)

	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "error opening config db")
	}

	dbState, err := isdb.NewDb(dbStateFn)

	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "error opening state db")
	}

	err = isdb.RunMigrations(dbData)

	if err != nil {
		log.Println("Error running migrations: ", err)
	}

	err = dbConfig.ReadConfig(config)

	if err != nil {
		if err == bolthold.ErrNotFound {
			log.Println("config not found -- try reading from old db")
			err := dbData.ReadConfig(config)
			if err != nil {
				log.Println("config not found in old db -- start with blank config")
			} else {
				err := dbConfig.WriteConfig(config)
				if err != nil {
					log.Println("Error writing config to new db", err)
				}
			}
		} else {
			log.Println("Error reading config, resetting: ", err)
			err := dbConfig.ResetDb()
			if err != nil {
				log.Println("Error resetting config db: ", err)
			}
		}
	}

	err = dbState.ReadState(state)

	if err != nil {
		if err == bolthold.ErrNotFound {
			log.Println("state not found -- try reading from old db")
			err := dbData.ReadState(state)
			if err != nil {
				log.Println("state not found in old db -- start with blank config")
			} else {
				err := dbState.WriteState(state)
				if err != nil {
					log.Println("Error writing state to new db", err)
				}
			}
		} else {
			log.Println("Error reading state, resetting: ", err)
			err := dbState.ResetDb()
			if err != nil {
				log.Println("Error resetting state db: ", err)
			}
		}
	}

	config.Init(state)

	return dbConfig, dbState, dbData, nil
}
