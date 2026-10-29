package data

import (
	"crypto/rand"
	"encoding/base32"
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
