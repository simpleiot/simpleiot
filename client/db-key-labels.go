package client

import (
	"strings"
)

// A point key written by the Prometheus scrape in the metrics client is a
// rendered label set: name=value pairs joined by commas, sorted by name. That
// is the shape SIOT's point model allows, but not the shape a time series
// database wants -- a query has to pull a label back out of the key with
// label_replace, and histogram_quantile cannot read an le it has to extract at
// all.
//
// Since the metrics client wrote the key to a grammar, the db client can read
// it back and write each label as its own database tag, which
// VictoriaMetrics stores as an ordinary label. The key tag is still written
// alongside, so a query that selects on the whole set keeps working.

// expandKeyLabels reads a point key as a label set and returns the labels it
// carried. A key that is not a label set returns nil, which the caller treats
// as "add nothing".
//
// The parse is deliberately strict and all or nothing, so a key is never half
// interpreted and every key the rest of SIOT writes -- eth0, /dev/sda, cpu0,
// an empty key stored as "0" -- is declined. It is also what handles a label
// value that held a comma: the key cannot be split on commas cleanly, the
// strict rule declines it, and the key tag still carries the whole set.
func expandKeyLabels(key string) map[string]string {
	if key == "" || key == "0" || !strings.Contains(key, "=") {
		return nil
	}

	chunks := strings.Split(key, ",")
	labels := make(map[string]string, len(chunks))

	for _, c := range chunks {
		name, val, ok := strings.Cut(c, "=")
		if !ok || !validLabelName(name) {
			return nil
		}

		if val == "" {
			// Prometheus treats an empty label and an absent label as the
			// same thing, so nothing is written for one
			continue
		}

		labels[name] = restoreNumericLabel(name, val)
	}

	if len(labels) == 0 {
		return nil
	}

	return labels
}

// validLabelName reports whether s is a Prometheus label name,
// [a-zA-Z_][a-zA-Z0-9_]*. Anything else means the key was not written as a
// label set.
func validLabelName(s string) bool {
	if s == "" {
		return false
	}

	for i := 0; i < len(s); i++ {
		c := s[i]

		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c == '_':
		case c >= '0' && c <= '9' && i > 0:
		default:
			return false
		}
	}

	return true
}

// restoreNumericLabel puts back the period SubjectSafeToken replaced in a
// label a query reads as a number.
//
// The substitution is not reversible in general -- an underscore in a stored
// key may have been an underscore to begin with -- so restoring every one
// would corrupt a version string or a path. Prometheus defines exactly two
// labels that carry a number, le and quantile, and those are the two that make
// histogram_quantile work, so only they are restored, and only when the value
// is otherwise all digits. A value such as le=+Inf needs no restoration and
// gets none.
func restoreNumericLabel(name, val string) string {
	if name != "le" && name != "quantile" {
		return val
	}

	for i := 0; i < len(val); i++ {
		if c := val[i]; (c < '0' || c > '9') && c != '_' {
			return val
		}
	}

	return strings.ReplaceAll(val, "_", ".")
}
