package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// Authorizer defines a mechanism needed to authorize stuff
type Authorizer interface {
	NewToken(id string) (string, error)
	Valid(req *http.Request) (bool, string)
}

// AlwaysValid is used to disable authentication
type AlwaysValid struct{}

// NewToken stub
func (AlwaysValid) NewToken(_ string) (string, error) { return "valid", nil }

// Valid stub
func (AlwaysValid) Valid(*http.Request) (bool, string) {
	return true, ""
}

// Key provides a key for signing authentication tokens.
type Key struct {
	bytes []byte
}

// NewKey returns a new Key of the given size.
func NewKey(bytes []byte) (key Key, err error) {
	key.bytes = bytes
	return
}

// NewToken returns a new authentication token signed by the Key.
func (k Key) NewToken(userID string) (string, error) {
	// FIXME Id is probably not the proper place to put the userid
	// but works for now
	claims := jwt.StandardClaims{
		ExpiresAt: time.Now().Add(168 * time.Hour).Unix(),
		Issuer:    "simpleiot",
		Id:        userID,
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString(k.bytes)
}

// ValidToken returns whether the given string
// is an authentication token signed by the Key.
func (k Key) ValidToken(str string) (bool, string) {
	userID, _, ok := k.TokenClaims(str)
	return ok, userID
}

// TokenClaims returns the user ID and expiry carried by a token signed by
// the Key. ok is false for a token that is not signed by the Key, has
// expired, or carries no user ID.
func (k Key) TokenClaims(str string) (userID string, expires time.Time, ok bool) {
	token, err := jwt.Parse(str, k.keyFunc)
	if err != nil || !token.Valid || token.Method.Alg() != "HS256" {
		return "", time.Time{}, false
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", time.Time{}, false
	}
	userID, ok = claims["jti"].(string)
	if !ok || userID == "" {
		return "", time.Time{}, false
	}
	if exp, ok := claims["exp"].(float64); ok {
		expires = time.Unix(int64(exp), 0)
	}
	return userID, expires, true
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
