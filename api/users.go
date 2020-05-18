package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/simpleiot/simpleiot/data"
)

// Users handles user requests.
type Users struct {
	db        *Db
	validator RequestValidator
}

// NewUsersHandler returns a new handler for user requests.
func NewUsersHandler(db *Db, v RequestValidator) Users {
	return Users{db: db, validator: v}
}

// ServeHTTP serves user requests.
func (u Users) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	validUser, userID := u.validator.Valid(req)
	if !validUser {
		http.Error(res, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	// only allow requests if user is part of root org
	isRoot, err := checkUserIsRoot(u.db.store, userUUID)

	if !isRoot {
		res.Write([]byte("[]"))
		return
	}

	var id string
	id, req.URL.Path = ShiftPath(req.URL.Path)

	if id == "" {
		switch req.Method {
		case http.MethodGet:
			email := req.URL.Query().Get("email")
			// get all users
			if email != "" {
				user, err := userByEmail(u.db.store, email)
				if err != nil {
					http.Error(res, err.Error(), http.StatusNotFound)
					return
				}
				encode(res, user)
				return
			}

			users, err := users(u.db.store)
			if err != nil {
				http.Error(res, err.Error(), http.StatusNotFound)
				return
			}
			if len(users) > 0 {
				encode(res, users)
			} else {
				res.Write([]byte("[]"))
			}
			return

		case http.MethodPost:
			// create user
			u.insertUser(res, req)
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
		// get a single user
		user, err := userByID(u.db.store, id)
		if err != nil {
			http.Error(res, err.Error(), http.StatusNotFound)
			return
		}
		encode(res, user)
		return

	case http.MethodPost:
		// update a single user
		u.updateUser(id, res, req)
		return

	case http.MethodDelete:
		err := deleteUser(u.db.store, idUUID)
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

func decode(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}

func encode(w io.Writer, v interface{}) error {
	return json.NewEncoder(w).Encode(v)
}

func (u Users) insertUser(res http.ResponseWriter, req *http.Request) {
	var user data.User
	if err := decode(req.Body, &user); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := insertUser(u.db.store, user)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	encode(res, data.StandardResponse{Success: true, ID: id})
}

func (u Users) updateUser(ID string, res http.ResponseWriter, req *http.Request) {
	var user data.User
	if err := decode(req.Body, &user); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	var err error
	user.ID, err = uuid.Parse(ID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusRequestedRangeNotSatisfiable)
		return
	}

	if err := updateUser(u.db.store, user); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	encode(res, data.StandardResponse{Success: true, ID: user.ID.String()})
}
