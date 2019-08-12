package isdb

import (
	"log"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// RunMigrations runs all database migrations
func RunMigrations(db *IsDb) error {
	faults, err := db.ReadFaultHistOld()
	if err != nil {
		log.Println("Error reading fault history during db migration")
	}

	for _, f := range faults {
		sampType := f.Fault.ToSampleType()
		if sampType != "" {
			s := data.Sample{
				Time: f.Time,
				Type: sampType,
			}

			err := db.WriteSample(s)
			if err != nil {
				log.Println("Error writing sample to db during migration", err)
			}
		}
	}

	err = db.store.DeleteMatching(isdata.Fault{}, nil)

	if err != nil {
		log.Println("Error clearing isdata.Fault types from database")
	}

	return nil
}
