package db

import "github.com/timshannon/bolthold"

// The SIOT project uses two types of stores:
//  - general database (users, groups, etc)
//  - time series db (samples)

// Db contains implementations of the stores
// we are suing.
type Db struct {
	store  *bolthold.Store
	influx *Influx
}

// General describes a general db store
type General interface {
}

// Timeseries provides an interface to a
// timeseries store to store samples.
type Timeseries interface {
}
