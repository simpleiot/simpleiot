# Time storage/format considerations

- Author: Cliff Brake, last updated: 2022-10-15
- PR/Discussion:
- Status: discussion

## Problem

How can we store timestamps that are:

- efficient
- high resolution (ns)
- portable
- won't run out of time values any time soon

We have multiple domains:

- Go
- MCU code
- Browser (ms resolution)
- SQLite
- Protbuf

Two questions:

- How should we store timestamps in SQLite?
- How should we transfer timestamps over the wire (typically protobuf)?

## Context

We currently use Go timestamps in Go code, and protobuf timestamps on the wire.

### Reference/Research

Browsers limit time resolution to MS for
[security reasons](https://community.tmpdir.org/t/high-rate-data-example-of-go-concurrency/654/4?u=cbrake).

2 ^ 64 nanoseconds is roughly ~ 584.554531 years.

https://github.com/jbenet/nanotime

For NTP time, the 64bits are broken in to seconds and fraction of seconds. The
top 32 bits is the seconds. The bottom 32 bits is the fraction of seconds. You
get the fraction by dividing the fraction part by 2^32.

[Windows uses](https://learn.microsoft.com/en-us/windows/win32/api/minwinbase/ns-minwinbase-filetime)
a 64-bit value representing the number of 100-nanosecond intervals since January
1, 1601 (UTC).

The Go Time type is fairly intelligent as it uses Montonic time when possible
and falls back to wall clock time when needed:

https://pkg.go.dev/time

> If Times t and u both contain monotonic clock readings, the operations
> t.After(u), t.Before(u), t.Equal(u), and t.Sub(u) are carried out using the
> monotonic clock readings alone, ignoring the wall clock readings. If either t
> or u contains no monotonic clock reading, these operations fall back to using
> the wall clock readings.

The Go Time type is fairly clever:

```go
type Time struct {
        // wall and ext encode the wall time seconds, wall time nanoseconds,
        // and optional monotonic clock reading in nanoseconds.
        //
        // From high to low bit position, wall encodes a 1-bit flag (hasMonotonic),
        // a 33-bit seconds field, and a 30-bit wall time nanoseconds field.
        // The nanoseconds field is in the range [0, 999999999].
        // If the hasMonotonic bit is 0, then the 33-bit field must be zero
        // and the full signed 64-bit wall seconds since Jan 1 year 1 is stored in ext.
        // If the hasMonotonic bit is 1, then the 33-bit field holds a 33-bit
        // unsigned wall seconds since Jan 1 year 1885, and ext holds a
        // signed 64-bit monotonic clock reading, nanoseconds since process start.
        wall uint64
        ext  int64

        // loc specifies the Location that should be used to
        // determine the minute, hour, month, day, and year
        // that correspond to this Time.
        // The nil location means UTC.
        // All UTC times are represented with loc==nil, never loc==&utcLoc.
        loc *Location
}
```

The Protbuf time format also has sec/ns sections:

```
message Timestamp {
  // Represents seconds of UTC time since Unix epoch
  // 1970-01-01T00:00:00Z. Must be from 0001-01-01T00:00:00Z to
  // 9999-12-31T23:59:59Z inclusive.
  int64 seconds = 1;

  // Non-negative fractions of a second at nanosecond resolution. Negative
  // second values with fractions must still have non-negative nanos values
  // that count forward in time. Must be from 0 to 999,999,999
  // inclusive.
  int32 nanos = 2;
}
```

## Decision

what was decided.

objections/concerns

## Consequences

what is the impact, both negative and positive.
