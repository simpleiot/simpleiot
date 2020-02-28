package api

import (
	"fmt"
	"net/http"

	"github.com/simpleiot/simpleiot/db"
)

// Auth handles user authentication requests.
type Auth struct {
	db  *db.Db
	key NewTokener
}

// NewTokener provides a new authentication token.
type NewTokener interface {
	NewToken() (string, error)
}

// NewAuthHandler returns a new authentication handler using the given key.
func NewAuthHandler(db *db.Db, key NewTokener) Auth {
	return Auth{db: db, key: key}
}

func (auth Auth) validLogin(email, password string) (bool, error) {
	users, err := auth.db.Users()
	if err != nil {
		return false, fmt.Errorf("error retrieving user list: %v", err)
	}

	for _, user := range users {
		if user.Email == email && user.Pass == password {
			return true, nil
		}
	}

	return false, nil
}

// ServeHTTP serves requests to authenticate.
func (auth Auth) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(res, "only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	email := req.FormValue("email")
	password := req.FormValue("password")

	if valid, err := auth.validLogin(email, password); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	} else if !valid {
		http.Error(res, "invalid login", http.StatusForbidden)
		return
	}

	token, err := auth.key.NewToken()
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.Write([]byte(token))
}
