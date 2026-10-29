package data

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
)

// idEncoding is lowercase Crockford base32. It has no "_", which separates
// the IDs in a stream name, no uppercase, so stream directories cannot
// collide on a case-insensitive filesystem, and no "i", "l", "o", or "u",
// so IDs are easier to read and copy.
var idEncoding = base32.NewEncoding("0123456789abcdefghjkmnpqrstvwxyz").
	WithPadding(base32.NoPadding)

// NewID returns a new random ID for a node or an instance: 128 random bits
// as 26 characters of lowercase Crockford base32. The 2 bits left over in
// the last character are zero, so it is one of "048cgmrw". IDs are opaque,
// so nothing decodes them, and older trees keep their UUIDs.
func NewID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return idEncoding.EncodeToString(b[:])
}

// maxIDLen bounds an ID chosen by hand. Generated IDs are 26 characters and
// UUIDs 36.
const maxIDLen = 64

// ValidID reports whether an ID chosen by hand, such as an instance ID or
// a node ID supplied through the API, is safe to use: 1 to 64 ASCII
// letters, digits, and "-", not starting with "-". Other characters break
// the system: "." and space split a NATS subject, "*" and ">" are
// wildcards, and "_" separates the two IDs in a stream name.
func ValidID(id string) error {
	if id == "" {
		return errors.New("ID is empty")
	}
	if len(id) > maxIDLen {
		return fmt.Errorf("ID %q is longer than %v characters", id, maxIDLen)
	}
	if id[0] == '-' {
		return fmt.Errorf("ID %q starts with \"-\"", id)
	}
	for _, c := range id {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-':
		default:
			return fmt.Errorf("ID %q contains %q; only letters, digits, and \"-\" are allowed", id, c)
		}
	}
	return nil
}
