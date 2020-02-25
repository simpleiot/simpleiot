package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/simpleiot/simpleiot/db"

	"github.com/dgrijalva/jwt-go"
)

// Auth handles user authentication requests.
type Auth struct {
	db  *db.Db
	key []byte
}

func NewAuthHandler(db *db.Db, key []byte) Auth {
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

	claims := jwt.StandardClaims{
		ExpiresAt: time.Now().Add(15 * time.Minute).Unix(),
		Issuer:    "simpleiot",
	}
	str, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(auth.key)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.Write([]byte(str))
}

func (auth Auth) ValidToken(str string) bool {
	token, err := jwt.Parse(str, auth.keyFunc)
	return err == nil &&
		token.Method.Alg() == "HS256" &&
		token.Valid
}

func (auth Auth) keyFunc(*jwt.Token) (interface{}, error) {
	return auth.key, nil
}
