package data

import (
	"fmt"
	"strings"
)

// invalidSubjectChars are the characters that cannot appear in a point type or
// key. A point is published on a subject ending in its type and key, so any of
// these would either split the subject into more tokens than a listener expects
// or make it an invalid NATS subject.
//
//	.  the subject token separator
//	*  and > the NATS wildcards, which would make the token unmatchable
//	whitespace, which NATS does not allow in a subject at all
const invalidSubjectChars = ". \t\r\n*>"

// CheckSubjectTokens returns an error if the point type or key contains a
// character that is not allowed in a NATS subject token.
//
// Points travel on subjects built from their type and key -- see
// client.SendPoints -- and listeners read the node ID and other routing
// information from fixed positions in the subject. A period in a type or key
// adds a token and shifts everything after it, so the point is delivered to
// the wrong handler. The store rejects such points on the way in, which keeps
// every subject the system publishes well formed.
func (p Point) CheckSubjectTokens() error {
	if i := strings.IndexAny(p.Type, invalidSubjectChars); i >= 0 {
		return fmt.Errorf(
			"point type %q contains %q, which is not allowed in a point type",
			p.Type, p.Type[i:i+1],
		)
	}

	if i := strings.IndexAny(p.Key, invalidSubjectChars); i >= 0 {
		return fmt.Errorf(
			"point (type %v) key %q contains %q, which is not allowed in a point key",
			p.Type, p.Key, p.Key[i:i+1],
		)
	}

	return nil
}

// SubjectSafeToken replaces every character that is not allowed in a point type
// or key with an underscore. It is for callers that generate keys from names
// they do not control, such as sysfs device names or network interface names.
// Data a device sends is never rewritten -- that is rejected instead, so the
// sender can be fixed.
func SubjectSafeToken(s string) string {
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(invalidSubjectChars, r) {
			return '_'
		}
		return r
	}, s)
}
