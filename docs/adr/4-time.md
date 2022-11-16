# Time storage/format considerations

- Author: Cliff Brake, last updated: 2022-10-15
- PR/Discussion:
- Status: discussion

**Contents**

<!-- toc -->

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

#### Browsers

Browsers limit time resolution to MS for
[security reasons](https://community.tmpdir.org/t/high-rate-data-example-of-go-concurrency/654/4?u=cbrake).

#### 64-bit nanoseconds

2 ^ 64 nanoseconds is roughly ~ 584.554531 years.

[https://github.com/jbenet/nanotime](https://github.com/jbenet/nanotime)

#### NTP

For NTP time, the 64bits are broken in to seconds and fraction of seconds. The
top 32 bits is the seconds. The bottom 32 bits is the fraction of seconds. You
get the fraction by dividing the fraction part by 2^32.

#### Linux

64-bit Linux systems are using 64bit timestamps (time_t) for seconds, and 32-bit
systems are switching to 64-bit to avoid the 2038 bug.

- https://musl.libc.org/time64.html
- https://sourceware.org/pipermail/libc-alpha/2022-November/143386.html

The Linux `clock_gettime()` function uses the following datatypes:

```
struct timeval {
	time_t          tv_sec;
	suseconds_t     tv_usec;
};
```

```
struct timespec {
	time_t          tv_sec;
	long            tv_nsec;
};
```

#### Windows

[Windows uses](https://learn.microsoft.com/en-us/windows/win32/api/minwinbase/ns-minwinbase-filetime)
a 64-bit value representing the number of 100-nanosecond intervals since January
1, 1601 (UTC).

#### Go

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

Go provides a [UnixNano()](https://pkg.go.dev/time#Time.UnixNano) function that
convers a Timestamp to nanoseconds elapsed since January 1, 1970 UTC.

To go the other way, Go provides a
[UnixMicro()](https://pkg.go.dev/time#UnixMicro) function to convert
microseconds since 1970 to a timestamp. The
[source code](https://cs.opensource.google/go/go/+/refs/tags/go1.19.2:src/time/time.go;l=1390)
could probably be modified to create a `UnixNano()` function.

```go
// UnixMicro returns the local Time corresponding to the given Unix time,
// usec microseconds since January 1, 1970 UTC.
func UnixMicro(usec int64) Time {
	return Unix(usec/1e6, (usec%1e6)*1e3)
}

// Unix returns the local Time corresponding to the given Unix time,
// sec seconds and nsec nanoseconds since January 1, 1970 UTC.
// It is valid to pass nsec outside the range [0, 999999999].
// Not all sec values have a corresponding time value. One such
// value is 1<<63-1 (the largest int64 value).
func Unix(sec int64, nsec int64) Time {
	if nsec < 0 || nsec >= 1e9 {
		n := nsec / 1e9
		sec += n
		nsec -= n * 1e9
		if nsec < 0 {
			nsec += 1e9
			sec--
		}
	}
	return unixTime(sec, int32(nsec))
}

```

#### Protobuf

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

#### MQTT

Note sure yet if MQTT defines a timestamp format.

Sparkplug does:

> timestamp
>
> - This is the timestamp in the form of an unsigned 64-bit integer representing
>   the number of milliseconds since epoch (Jan 1, 1970). It is highly
>   recommended that this time is in UTC. This timestamp is meant to represent
>   the time at which the message was published

#### CRDTs

LWW (last write wins) CRDTs often use a logical clock.
[crsql](https://github.com/vlcn-io/cr-sqlite) uses a 64-bit logical clock.

### Do we need nanosecond resolution?

Many IoT systems only support MS resolution. However, this is sometimes cited as
a deficiency in applications where higher resolution is needed (e.g. power
grid).

## Decision

what was decided.

objections/concerns

## Consequences

what is the impact, both negative and positive.
