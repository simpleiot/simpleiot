package api

import (
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/simpleiot/simpleiot/data"
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
			orgs, err := orgs(o.db.store)
			if err != nil {
				http.Error(res, err.Error(), http.StatusNotFound)
				return
			}
			encode(res, orgs)
			return

		case http.MethodPost:
			// create user
			o.insertOrg(res, req)
			return

		default:
			http.Error(res, "invalid method", http.StatusMethodNotAllowed)
			return
		}
	}

	idUUID, err := uuid.Parse(id)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	switch req.Method {
	case http.MethodGet:
		// get a single org

		org, err := org(o.db.store, idUUID)
		if err != nil {
			http.Error(res, err.Error(), http.StatusNotFound)
			return
		}
		encode(res, org)
		return

	case http.MethodPost:
		// update a single org
		o.updateOrg(id, res, req)
		return
	}

	http.Error(res, "invalid method", http.StatusMethodNotAllowed)
}

func (o Orgs) insertOrg(res http.ResponseWriter, req *http.Request) {
	var org data.Org
	if err := decode(req.Body, &org); err != nil {
		log.Println("Error decoding org: ", err)
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := insertOrg(o.db.store, org)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	encode(res, data.StandardResponse{Success: true, ID: id})
}

func (o Orgs) updateOrg(ID string, res http.ResponseWriter, req *http.Request) {
	var org data.Org
	if err := decode(req.Body, &org); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	var err error
	org.ID, err = uuid.Parse(ID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusRequestedRangeNotSatisfiable)
		return
	}

	if err := updateOrg(o.db.store, org); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	encode(res, data.StandardResponse{Success: true, ID: org.ID.String()})
}
