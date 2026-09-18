package api

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func TestTokenClaims(t *testing.T) {
	k, _ := NewKey([]byte("0123456789abcdef0123456789abcdef"))

	tok, err := k.NewToken("U")
	if err != nil {
		t.Fatal(err)
	}
	if id, _, ok := k.TokenClaims(tok); !ok || id != "U" {
		t.Fatalf("own token: %v %v", id, ok)
	}

	sign := func(claims jwt.StandardClaims, key []byte) string {
		s, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(key)
		return s
	}
	future := time.Now().Add(time.Hour).Unix()

	for _, tt := range []struct {
		name string
		tok  string
	}{
		{"other issuer", sign(jwt.StandardClaims{ExpiresAt: future, Issuer: "someone", Id: "U"}, k.bytes)},
		{"no issuer", sign(jwt.StandardClaims{ExpiresAt: future, Id: "U"}, k.bytes)},
		{"other key", sign(jwt.StandardClaims{ExpiresAt: future, Issuer: tokenIssuer, Id: "U"}, []byte("x"))},
		{"expired", sign(jwt.StandardClaims{ExpiresAt: time.Now().Add(-time.Hour).Unix(), Issuer: tokenIssuer, Id: "U"}, k.bytes)},
		{"no user", sign(jwt.StandardClaims{ExpiresAt: future, Issuer: tokenIssuer}, k.bytes)},
	} {
		if _, _, ok := k.TokenClaims(tt.tok); ok {
			t.Errorf("%v: accepted", tt.name)
		}
	}
}

func TestConnectHasToken(t *testing.T) {
	for _, tt := range []struct {
		msg  string
		want bool
	}{
		{`CONNECT {"auth_token":"tok","lang":"nats.ws"}` + "\r\n", true},
		{`connect {"auth_token":"tok"}`, true},
		{`CONNECT {"user":"U","pass":"eyJ.x.y"}` + "\r\n", false},
		{`CONNECT {"nkey":"U...","sig":"..."}`, false},
		{`CONNECT {"auth_token":""}`, false},
		{`PING` + "\r\n", false},
		{`CONNECT not json`, false},
		{``, false},
	} {
		if got := connectHasToken([]byte(tt.msg)); got != tt.want {
			t.Errorf("%q: got %v", tt.msg, got)
		}
	}
}
