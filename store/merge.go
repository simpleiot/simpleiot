package store

import "time"

// tipWins reports whether an incoming point replaces the current subject
// tip, applying the ADR-7 merge rule:
//
//   - the newest timestamp wins
//   - equal timestamps from different origins resolve to the lexically
//     greater origin ID, so instances merging the same streams
//     independently converge on the same tip
//   - an identical (timestamp, origin) delivery is a no-op, which makes
//     the merge idempotent when the same point arrives more than once
//     (Stage 3 delivers points both live and through replica streams)
//
// origin is the root node ID of the instance that wrote the point; an
// unknown origin sorts first, so a delivery with a known origin at the
// same timestamp replaces it.
func tipWins(curTime time.Time, curOrigin string, inTime time.Time, inOrigin string) bool {
	if inTime.After(curTime) {
		return true
	}
	if curTime.After(inTime) {
		return false
	}
	return inOrigin > curOrigin
}
