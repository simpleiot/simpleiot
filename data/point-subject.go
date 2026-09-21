package data

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// invalidSubjectChars are the characters that cannot appear in a point type or
// key. A point is published on a subject ending in its type and key, so any of
// these would either split the subject into more tokens than a listener expects
// or make it an invalid NATS subject.
//
//	.  the subject token separator
//	*  and > the NATS wildcards, which would make the token unmatchable
//	whitespace and NUL, which NATS does not allow in a subject at all
//
// Bytes that are not valid UTF-8 are rejected as well (see invalidTokenChar).
// NATS accepts both NUL and invalid UTF-8 on publish, but refuses them when it
// loads a stream's index, so a single such subject makes every restart rebuild
// the stream state from the message blocks.
const invalidSubjectChars = ". \t\r\n*>\x00"

// invalidTokenChar returns the first character in s that is not allowed in a
// subject token.
func invalidTokenChar(s string) (string, bool) {
	if i := strings.IndexAny(s, invalidSubjectChars); i >= 0 {
		return s[i : i+1], true
	}
	for i, r := range s {
		if r == utf8.RuneError {
			_, size := utf8.DecodeRuneInString(s[i:])
			return s[i : i+size], true
		}
	}
	return "", false
}

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
	if c, bad := invalidTokenChar(p.Type); bad {
		return fmt.Errorf(
			"point type %q contains %q, which is not allowed in a point type",
			p.Type, c,
		)
	}

	if c, bad := invalidTokenChar(p.Key); bad {
		return fmt.Errorf(
			"point (type %q) key %q contains %q, which is not allowed in a point key",
			p.Type, p.Key, c,
		)
	}

	return nil
}

// CheckSubjectToken returns an error if a string cannot be one token of a
// NATS subject: it is empty or contains a period, a wildcard, whitespace,
// NUL, or bytes that are not valid UTF-8. Node IDs travel in subjects, so an ID taken from a peer or a
// point (an enrolling device names its own, a rule action names its target)
// is checked with it; what names the ID in the error.
func CheckSubjectToken(what, s string) error {
	if s == "" {
		return fmt.Errorf("%v is empty", what)
	}
	if c, bad := invalidTokenChar(s); bad {
		return fmt.Errorf("%v %q contains %q, which is not allowed", what, s, c)
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
		if r == utf8.RuneError || strings.ContainsRune(invalidSubjectChars, r) {
			return '_'
		}
		return r
	}, s)
}
