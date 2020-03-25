package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/simpleiot/simpleiot/data"
)

// Users handles user requests.
type Users struct {
	db *Db
}

// NewUsersHandler returns a new handler for user requests.
func NewUsersHandler(db *Db) Users {
	return Users{db: db}
}

// ServeHTTP serves user requests.
func (u Users) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var id string
	id, req.URL.Path = ShiftPath(req.URL.Path)

	if id == "" {
		switch req.Method {
		case http.MethodGet:
			// get all users
			users, err := u.db.Users()
			if err != nil {
				http.Error(res, err.Error(), http.StatusNotFound)
				return
			}
			json.NewEncoder(res).Encode(users)
			return

		case http.MethodPost:
			// create user
			u.createUser(res, req)
			return

		default:
			http.Error(res, "invalid method", http.StatusMethodNotAllowed)
			return
		}
	}

	switch req.Method {
	case http.MethodGet:
		// get a single user
		user, err := u.db.User(id)
		if err != nil {
			http.Error(res, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(res).Encode(user)
		return

	}

	http.Error(res, "invalid method", http.StatusMethodNotAllowed)
}

func decode(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}

func encode(w io.Writer, v interface{}) error {
	return json.NewEncoder(w).Encode(v)
}

func (u Users) createUser(res http.ResponseWriter, req *http.Request) {
	var user data.User
	if err := decode(req.Body, &user); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := createUser(u.db, user)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}

	encode(res, data.StandardResponse{Success: true, ID: id})
}

func (u Users) upsertUser(res http.ResponseWriter, req *http.Request, id string) {
	var user data.User
	if err := decode(req.Body, &user); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	if err := u.db.UserUpsert(id, user); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}

	encode(res, data.StandardResponse{Success: true, ID: id})
}
