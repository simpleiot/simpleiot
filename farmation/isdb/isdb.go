package isdb

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/timshannon/bolthold"
	"go.etcd.io/bbolt"
)

// IsDb is used for all db access in the application.
// We will eventually turn this into an interface to
// handle multiple Db backends.
type IsDb struct {
	filename string
	store    *bolthold.Store
}

// NewDb creates a new Db instance for the app
func NewDb(fileName string) (*IsDb, error) {
	boltOptions := bbolt.Options{
		FreelistType: bbolt.FreelistArrayType,
	}

	options := bolthold.Options{
		Encoder: bolthold.DefaultEncode,
		Decoder: bolthold.DefaultDecode,
		Options: &boltOptions,
	}
	store, err := bolthold.Open(fileName, 0666, &options)
	if err != nil {
		return nil, err
	}

	return &IsDb{
		filename: fileName,
		store:    store,
	}, nil
}

// ReadConfig reads the IS config from the database
func (db *IsDb) ReadConfig(config *isdata.Config) error {
	err := db.store.Get(0, config)

	if err != nil {
		return err
	}

	return nil
}

// ResetConfig is used to reset the config
func (db *IsDb) ResetConfig() error {
	return db.store.Delete(0, isdata.Config{})
}

// ResetDb is used to reset the entire database (used if it is corrupted, etc).
func (db *IsDb) ResetDb() error {
	err := db.store.Close()
	if err != nil {
		log.Println("Error closing db", err)
	}
	err = os.Remove(db.filename)
	if err != nil {
		log.Println("error removing db file: ", err)
		return err
	}

	db.store, err = bolthold.Open(db.filename, 0666, nil)

	return err
}

// ReadState reads the IS config from the database
func (db *IsDb) ReadState(state *isdata.State) error {
	err := db.store.Get(0, state)

	if err != nil {
		return err
	}

	return nil
}

// ReadSamples reads samples from the database
// Samples are flow, pressure, amount, etc.
// if callback returns error, this function returns with that error
func (db *IsDb) ReadSamples(start time.Time, callback func(s data.Sample) error) error {
	boltdb := db.store.Bolt()

	key, err := bolthold.DefaultEncode(start)
	if err != nil {
		return err
	}

	err = boltdb.View(func(tx *bbolt.Tx) error {
		// Assume bucket exists and has keys
		c := tx.Bucket([]byte("Sample")).Cursor()

		for k, v := c.Seek(key); k != nil; k, v = c.Next() {
			t := time.Time{}
			err := bolthold.DefaultDecode(k, &t)
			if err != nil {
				return err
			}
			s := data.Sample{}
			err = bolthold.DefaultDecode(v, &s)
			if err != nil {
				return err
			}
			s.Time = t
			err = callback(s)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

// ReadFaultHistOld reads legacy fault data from db
// used only for migrations
func (db *IsDb) ReadFaultHistOld() (isdata.Faults, error) {
	var faults isdata.Faults

	err := db.store.Find(&faults, nil)

	if err != nil {
		if err == bolthold.ErrNotFound {
			// data is not stored, so simply return nil array and nil error
			return nil, nil
		}

		// there was an error reading so return nil array and error
		return nil, err
	}

	return faults, nil
}

// ReadFaultHist reads the IS system fault history from the database
// Only iterates through database for x most recent faults. Set x to
// a negative integer to read all faults
func (db *IsDb) ReadFaultHist(start time.Time) ([]data.Sample, error) {
	var faults []data.Sample
	boltdb := db.store.Bolt()

	fmt.Println("CLIFF: start: ", start)

	key, err := bolthold.DefaultEncode(start)
	if err != nil {
		return faults, err
	}

	isFault := func(s data.Sample) bool {
		switch s.Type {
		case isdata.SampleTypeFaultFlowOff,
			isdata.SampleTypeFaultPresLow,
			isdata.SampleTypeFaultShutdown,
			isdata.SampleTypeFaultPresHigh,
			isdata.SampleTypeFaultNtFlowOff,
			isdata.SampleTypeFaultNtPresLow,
			isdata.SampleTypeFaultNtPresHigh:
			return true
		}

		return false
	}

	boltdb.View(func(tx *bbolt.Tx) error {
		// Assume bucket exists and has keys
		c := tx.Bucket([]byte("Sample")).Cursor()

		for k, v := c.Seek(key); k != nil; k, v = c.Next() {
			t := time.Time{}
			err := bolthold.DefaultDecode(k, &t)
			if err != nil {
				return err
			}
			//fmt.Println("cliff: t: ", t)
			s := data.Sample{}
			err = bolthold.DefaultDecode(v, &s)
			if err != nil {
				return err
			}
			if isFault(s) {
				s.Time = t
				faults = append(faults, s)
			}
		}

		return nil
	})

	return faults, nil
}

// WriteConfig writes the IS config to the database
func (db *IsDb) WriteConfig(config *isdata.Config) error {

	return db.store.Upsert(0, config)
}

// WriteState writes the IS state to the database
func (db *IsDb) WriteState(state *isdata.State) error {

	return db.store.Upsert(0, state)
}

// DataMeta is used to store meta information about data in the database
type DataMeta struct {
	SampleCount int
}

// WriteSample writes a sample to the database
// Samples are flow, pressure, amount, etc.
func (db *IsDb) WriteSample(sample data.Sample) error {
	dataMeta := DataMeta{}
	err := db.store.Get(0, &dataMeta)
	if err != nil {
		// attempt to init metadata
		_, err = db.GetSampleCount()
		if err != nil {
			return err
		}
	}
	err = db.store.Insert(sample.Time, sample)
	if err != nil {
		return err
	}

	dataMeta.SampleCount++
	return db.store.Upsert(0, &dataMeta)
}

// GetSampleCount from database. Warning, this function
// is very inefficient as it appears to simply iterate through
// all samples.
func (db *IsDb) GetSampleCount() (int, error) {
	dataMeta := DataMeta{}
	err := db.store.Get(0, &dataMeta)
	if err != nil {
		err = db.store.Upsert(0, &dataMeta)
		if err != nil {
			return 0, err
		}
	}

	return dataMeta.SampleCount, nil
}

// WriteFaultHist writes the system fault history to the database
func (db *IsDb) WriteFaultHist(fault isdata.Fault) error {
	return db.store.Insert(time.Now(), fault)
}
