package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/db"
)

// Users handles user requests.
type Users struct {
	db        *db.Db
	validator RequestValidator
}

// NewUsersHandler returns a new handler for user requests.
func NewUsersHandler(db *db.Db, v RequestValidator) Users {
	return Users{db: db, validator: v}
}

// ServeHTTP serves user requests.
func (h Users) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	validUser, userID := h.validator.Valid(req)
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
	isRoot, err := h.db.UserIsRoot(userUUID)

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
				user, err := h.db.UserByEmail(email)
				if err != nil {
					http.Error(res, err.Error(), http.StatusNotFound)
					return
				}
				encode(res, user)
				return
			}

			users, err := h.db.Users()
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
			h.insertUser(res, req)
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
		user, err := h.db.UserByID(id)
		if err != nil {
			http.Error(res, err.Error(), http.StatusNotFound)
			return
		}
		encode(res, user)
		return

	case http.MethodPost:
		// update a single user
		h.updateUser(id, res, req)
		return

	case http.MethodDelete:
		err := h.db.UserDelete(idUUID)
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

func (h Users) insertUser(res http.ResponseWriter, req *http.Request) {
	var user data.User
	if err := decode(req.Body, &user); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := h.db.UserInsert(user)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	encode(res, data.StandardResponse{Success: true, ID: id})
}

func (h Users) updateUser(ID string, res http.ResponseWriter, req *http.Request) {
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

	if err := h.db.UserUpdate(user); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	encode(res, data.StandardResponse{Success: true, ID: user.ID.String()})
}
