package api

import (
	"net/http"
)

// Orgs handles org requests.
type Orgs struct {
	db *Db
}

// NewOrgsHandler returns a new handler for org requests.
func NewOrgsHandler(db *Db) Orgs {
	return Orgs{db: db}
}

// ServeHTTP serves org requests.
func (o Orgs) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var id string
	id, req.URL.Path = ShiftPath(req.URL.Path)

	if id == "" {
		switch req.Method {
		case http.MethodGet:
			// get all orgs
			// TODO: get user orgs
			orgs, err := orgs(o.db.store)
			if err != nil {
				http.Error(res, err.Error(), http.StatusNotFound)
				return
			}
			encode(res, orgs)
			return

		default:
			http.Error(res, "invalid method", http.StatusMethodNotAllowed)
			return
		}
	}

	http.Error(res, "invalid method", http.StatusMethodNotAllowed)
}
