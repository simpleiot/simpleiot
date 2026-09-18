package api

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// tokenIssuer is the issuer claim on every user token; a token that
// names another issuer is refused.
const tokenIssuer = "simpleiot"

// TokenLifetime is how long a sign-in token is valid. A token is also
// refused as soon as the user it names is gone from the tree, and the NATS
// server closes a browser connection when its token expires.
const TokenLifetime = 168 * time.Hour

// UserAuthority is what the HTTP API needs from the store to authenticate
// and scope a request from a signed-in user: who a token belongs to, where
// that user sits in the tree, and whether a node is inside one of those
// places.
type UserAuthority interface {
	UserFromToken(token string) (userID string, expires time.Time, ok bool)
	UserAnchors(userID string) []string
	IsUnder(id, anchor string) bool
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
	claims := jwt.StandardClaims{
		ExpiresAt: time.Now().Add(TokenLifetime).Unix(),
		Issuer:    tokenIssuer,
		Id:        userID,
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString(k.bytes)
}

// TokenClaims returns the user ID and expiry carried by a token signed by
// the Key. ok is false for a token that is not signed by the Key, uses
// another algorithm, names another issuer, has expired, or carries no user
// ID.
func (k Key) TokenClaims(str string) (userID string, expires time.Time, ok bool) {
	token, err := jwt.Parse(str, k.keyFunc)
	if err != nil || !token.Valid || token.Method.Alg() != "HS256" {
		return "", time.Time{}, false
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !claims.VerifyIssuer(tokenIssuer, true) {
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

func (k Key) keyFunc(*jwt.Token) (interface{}, error) {
	return k.bytes, nil
}
