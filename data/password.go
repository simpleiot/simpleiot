package data

import (
	"crypto/subtle"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// User passwords are stored as bcrypt hashes in the pass point on the user
// node. The store hashes locally-written pass points before persisting them,
// so the plaintext never lands in the store, exports, or sync streams. Values
// written before hashing existed are still plaintext; CheckPassword accepts
// them and reports that a rehash is needed, and the store rewrites the point
// as a hash after the next successful login.

// PasswordIsHashed reports whether a stored password value is a bcrypt hash.
func PasswordIsHashed(stored string) bool {
	return strings.HasPrefix(stored, "$2a$") ||
		strings.HasPrefix(stored, "$2b$") ||
		strings.HasPrefix(stored, "$2y$")
}

// HashPassword hashes a plaintext password for storage. bcrypt limits
// passwords to 72 bytes and returns an error for longer values.
func HashPassword(plain string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

// CheckPassword verifies a candidate password against the stored value, which
// may be a bcrypt hash or a legacy plaintext password. needsRehash is true
// when the check succeeded against a legacy value, meaning the stored
// password should be rewritten as a hash.
func CheckPassword(stored, candidate string) (ok, needsRehash bool) {
	if PasswordIsHashed(stored) {
		err := bcrypt.CompareHashAndPassword([]byte(stored), []byte(candidate))
		return err == nil, false
	}

	ok = subtle.ConstantTimeCompare([]byte(stored), []byte(candidate)) == 1
	return ok, ok
}
