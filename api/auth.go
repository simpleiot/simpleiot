package api

import (
	"net/http"

	"github.com/simpleiot/simpleiot/data"
)

// Auth handles user authentication requests.
type Auth struct {
	db  *Db
	key NewTokener
}

// NewTokener provides a new authentication token.
type NewTokener interface {
	NewToken() (string, error)
}

// NewAuthHandler returns a new authentication handler using the given key.
func NewAuthHandler(db *Db, key NewTokener) Auth {
	return Auth{db: db, key: key}
}

// ServeHTTP serves requests to authenticate.
func (auth Auth) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(res, "only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	email := req.FormValue("email")
	password := req.FormValue("password")

	priv, err := loginPrivilege(auth.db.store, email, password)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	switch priv {
	case none:
		http.Error(res, "invalid login", http.StatusForbidden)
		return
	}

	token, err := auth.key.NewToken()
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	encode(res, data.Auth{
		Privilege: string(priv),
		Token:     token,
	})
}
