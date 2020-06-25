package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/db"
)

// Groups handles group requests.
type Groups struct {
	db        *db.Db
	validator RequestValidator
}

// NewGroupsHandler returns a new handler for group requests.
func NewGroupsHandler(db *db.Db, v RequestValidator) Groups {
	return Groups{db: db, validator: v}
}

// ServeHTTP serves group requests.
func (h Groups) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	validUser, _ := h.validator.Valid(req)
	if !validUser {
		http.Error(res, "Unauthorized", http.StatusUnauthorized)
		return
	}

	/*
		userUUID, err := uuid.Parse(userID)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		// only allow requests if user is part of root group
		isRoot, err := checkUserIsRoot(o.db.store, userUUID)

		if !isRoot {
			res.Write([]byte("[]"))
			return
		}
	*/

	var id string
	id, req.URL.Path = ShiftPath(req.URL.Path)

	if id == "" {
		switch req.Method {
		case http.MethodGet:
			// get all groups
			groups, err := h.db.Groups()
			if err != nil {
				http.Error(res, err.Error(), http.StatusNotFound)
				return
			}
			if len(groups) > 0 {
				encode(res, groups)
			} else {
				res.Write([]byte("[]"))
			}
			return

		case http.MethodPost:
			// create user
			h.insertGroup(res, req)
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
		// get a single group

		group, err := h.db.Group(idUUID)
		if err != nil {
			http.Error(res, err.Error(), http.StatusNotFound)
			return
		}
		encode(res, group)
		return

	case http.MethodPost:
		// update a single group
		h.updateGroup(idUUID, res, req)
		return

	case http.MethodDelete:
		err := h.db.GroupDelete(idUUID)
		if err != nil {
			http.Error(res, err.Error(), http.StatusNotFound)
		} else {
			en := json.NewEncoder(res)
			en.Encode(data.StandardResponse{Success: true, ID: id})
		}
		return
	}

	http.Error(res, "invalid method", http.StatusMethodNotAllowed)
}

func (h Groups) insertGroup(res http.ResponseWriter, req *http.Request) {
	var group data.Group
	if err := decode(req.Body, &group); err != nil {
		log.Println("Error decoding group: ", err)
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := h.db.GroupInsert(group)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	encode(res, data.StandardResponse{Success: true, ID: id})
}

func (h Groups) updateGroup(ID uuid.UUID, res http.ResponseWriter, req *http.Request) {
	var group data.Group
	if err := decode(req.Body, &group); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	group.ID = ID

	if err := h.db.GroupUpdate(group); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	encode(res, data.StandardResponse{Success: true, ID: group.ID.String()})
}
