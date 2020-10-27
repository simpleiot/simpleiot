package api

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
)

// Authorizer defines a mechanism needed to authorize stuff
type Authorizer interface {
	NewToken(id string) (string, error)
	Valid(req *http.Request) (bool, string)
}

// AlwaysValid is used to disable authentication
type AlwaysValid struct{}

// NewToken stub
func (AlwaysValid) NewToken(id string) (string, error) { return "valid", nil }

// Valid stub
func (AlwaysValid) Valid(*http.Request) (bool, string) {
	return true, ""
}

// Key provides a key for signing authentication tokens.
type Key struct {
	bytes []byte
}

// NewKey returns a new Key of the given size.
func NewKey(size int) (key Key, err error) {
	var f *os.File
	f, err = os.Open("/dev/urandom")
	if err != nil {
		return
	}
	defer f.Close()
	key.bytes = make([]byte, size)
	_, err = f.Read(key.bytes)
	return
}

// NewToken returns a new authentication token signed by the Key.
func (k Key) NewToken(userID string) (string, error) {
	// FIXME Id is probably not the proper place to put the userid
	// but works for now
	claims := jwt.StandardClaims{
		ExpiresAt: time.Now().Add(30 * time.Minute).Unix(),
		Issuer:    "simpleiot",
		Id:        userID,
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString(k.bytes)
}

// ValidToken returns whether the given string
// is an authentication token signed by the Key.
func (k Key) ValidToken(str string) (bool, string) {
	token, err := jwt.Parse(str, k.keyFunc)
	if err != nil {
		return false, ""
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return false, ""
	}
	userID, ok := claims["jti"].(string)
	if !ok {
		return false, ""
	}
	return (err == nil &&
		token.Method.Alg() == "HS256" &&
		token.Valid), userID
}

// Valid returns whether the given request
// bears an authorization token signed by the Key.
func (k Key) Valid(req *http.Request) (bool, string) {
	fields := strings.Fields(req.Header.Get("Authorization"))
	if len(fields) < 2 {
		return false, ""
	}
	if fields[0] != "Bearer" {
		return false, ""
	}

	valid, userID := k.ValidToken(fields[1])
	return valid, userID
}

func (k Key) keyFunc(*jwt.Token) (interface{}, error) {
	return k.bytes, nil
}
