package isdb

import (
	"encoding/binary"
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
		Timeout:      5 * time.Second,
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

	err = store.Bolt().Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(sampleBucket))
		if err != nil {
			return fmt.Errorf("create sample bucket: %s", err)
		}
		return nil
	})

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

// DeleteSamples used to purse sample data
func (db *IsDb) DeleteSamples() error {
	boltdb := db.store.Bolt()

	return boltdb.Update(func(tx *bbolt.Tx) error {
		err := tx.DeleteBucket([]byte(sampleBucket))

		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists([]byte(sampleBucket))
		if err != nil {
			return fmt.Errorf("create sample bucket: %s", err)
		}

		dataMeta := DataMeta{}
		// reset sample count
		return db.store.TxUpsert(tx, 0, &dataMeta)
	})
}

// ReadSamples reads samples from the database
// Samples are flow, pressure, amount, etc.
// if callback returns error, this function returns with that error
func (db *IsDb) ReadSamples(cnt int, callback func(s data.Sample) error) error {
	boltdb := db.store.Bolt()

	return boltdb.View(func(tx *bbolt.Tx) error {
		// Assume bucket exists and has keys
		c := tx.Bucket([]byte(sampleBucket)).Cursor()

		count := 0

		// rewind 5000 records
		for k, _ := c.Last(); k != nil && count <= cnt; c.Prev() {
			count++
		}

		if count <= 0 {
			// no records
			return nil
		}

		// now replay forward to end of records
		for k, v := c.Next(); k != nil; k, v = c.Next() {
			tUnix := btoi(k)
			t := time.Unix(0, tUnix)

			s := data.Sample{}
			err := bolthold.DefaultDecode(v, &s)
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

	key := itob(start.UnixNano())

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

	err := boltdb.View(func(tx *bbolt.Tx) error {
		// Assume bucket exists and has keys
		c := tx.Bucket([]byte(sampleBucket)).Cursor()

		for k, v := c.Seek(key); k != nil; k, v = c.Next() {
			tUnix := btoi(k)
			t := time.Unix(0, tUnix)

			s := data.Sample{}
			err := bolthold.DefaultDecode(v, &s)
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

	if err != nil {
		return faults, err
	}

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

// itob returns an 8-byte big endian representation of v.
func itob(v int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

// btoi converts 8-byte big endian to int
func btoi(buf []byte) int64 {
	return int64(binary.BigEndian.Uint64(buf))
}

// create a unique name for buckets that we don't manage with bolthold
var sampleBucket = "CustSamples"

// WriteSample writes a sample to the database
// Samples are flow, pressure, amount, etc.
func (db *IsDb) WriteSample(sample data.Sample) error {
	return db.store.Bolt().Update(func(tx *bbolt.Tx) error {
		dataMeta := DataMeta{}
		err := db.store.TxGet(tx, 0, &dataMeta)
		if err != nil {
			// attempt to init metadata
			err = db.store.TxUpsert(tx, 0, &dataMeta)
			if err != nil {
				return err
			}
		}

		key := sample.Time.UnixNano()
		keyB := itob(key)

		bucket := tx.Bucket([]byte(sampleBucket))
		sampleE, err := bolthold.DefaultEncode(sample)
		if err != nil {
			return err
		}

		err = bucket.Put(keyB, sampleE)
		if err != nil {
			return err
		}

		dataMeta.SampleCount++
		return db.store.TxUpsert(tx, 0, &dataMeta)
	})
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
