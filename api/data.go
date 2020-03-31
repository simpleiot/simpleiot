package api

import (
	"github.com/simpleiot/simpleiot/data"
	"net/http"
)

// Data handles data requests.
type Data struct {
	db *Db
}

// NewDataHandler returns a new handler for data requests.
func NewDataHandler(db *Db) Data {
	return Data{db: db}
}

// ServeHTTP serves data requests.
func (d Data) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	// These all ought to be in one transaction.
	orgs, err := orgs(d.db.store)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}

	users, err := users(d.db.store)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}

	devices, err := devices(d.db.store)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}

	payload := data.Data{
		Orgs:    orgs,
		Users:   users,
		Devices: devices,
	}
	encode(res, payload)
}
