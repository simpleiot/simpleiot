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
	NewToken() (string, error)
	Valid(req *http.Request) bool
}

// AlwaysValid is used to disable authentication
type AlwaysValid struct{}

// NewToken stub
func (AlwaysValid) NewToken() (string, error) { return "valid", nil }

// Valid stub
func (AlwaysValid) Valid(*http.Request) bool { return true }

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
func (k Key) NewToken() (string, error) {
	claims := jwt.StandardClaims{
		ExpiresAt: time.Now().Add(15 * time.Minute).Unix(),
		Issuer:    "simpleiot",
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString(k.bytes)
}

// ValidToken returns whether the given string
// is an authentication token signed by the Key.
func (k Key) ValidToken(str string) bool {
	token, err := jwt.Parse(str, k.keyFunc)
	return err == nil &&
		token.Method.Alg() == "HS256" &&
		token.Valid
}

// Valid returns whether the given request
// bears an authorization token signed by the Key.
func (k Key) Valid(req *http.Request) bool {
	fields := strings.Fields(req.Header.Get("Authorization"))
	return len(fields) == 2 &&
		fields[0] == "Bearer" &&
		k.ValidToken(fields[1])
}

func (k Key) keyFunc(*jwt.Token) (interface{}, error) {
	return k.bytes, nil
}
