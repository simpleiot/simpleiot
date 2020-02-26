package api

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
)

type Key struct {
	bytes []byte
}

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

func (k Key) NewToken() (string, error) {
	claims := jwt.StandardClaims{
		ExpiresAt: time.Now().Add(15 * time.Minute).Unix(),
		Issuer:    "simpleiot",
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString(k.bytes)
}

func (k Key) ValidToken(str string) bool {
	token, err := jwt.Parse(str, k.keyFunc)
	return err == nil &&
		token.Method.Alg() == "HS256" &&
		token.Valid
}

func (k Key) ValidHeader(req *http.Request) bool {
	fields := strings.Fields(req.Header.Get("Authorization"))
	return len(fields) == 2 &&
		fields[0] == "Bearer" &&
		k.ValidToken(fields[1])
}

func (k Key) keyFunc(*jwt.Token) (interface{}, error) {
	return k.bytes, nil
}
