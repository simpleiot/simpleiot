# Time Validation

- Author: Cliff Brake
- PR/Discussion:
- Status: discussion

**Contents**

<!-- toc -->

## Problem

To date, SIOT has been deployed to systems with RTCs and solid network connections,
so time is fairly stable, thus this has not been a big concern. However, we are looking
to deploy to edge systems, some with cellular modem connections and some without a 
battery backed RTC, so they may boot without a valid time.

SIOT is very dependent on data having valid timestamps. If timestamps are not correct,
the following probems may occur:

- old data may be preferred over newer data in the point CRDT merge algorithm
- data stored in time series databases may have the wrong time stamps

Additionally, there are edge systems that don't have a real-time clock and 
power up with an invalid time until a NTP process gets the current time.

We may need some systems to operate (run rules, etc) without a valid network connection
(offline) and valid time.

## Context/Discussion

### Clients affected

- db (InfluxDB driver)
- sync (sends data upstream)
- store (not sure ???)

The db and sync clients should not process points (or perhaps buffer them until) until we 
are sure the system has a valid time. How does it get this information? Possibilities
include:

1. creating a broadcast or other special message subject that clients can optionally 
   listen to. Perhaps the NTP client can send this message.
  - syncronization may be a problem here if NTP client sends messages before a 
    client has started.
1. query for system state, and NTP sync status could be a field in this state.
  - should this be part of the root device node?
  - or a special hardcoded message?
  - it would be useful to track system state as a standard point so it gets
    syncronized and stored in influxdb, therefore as part of the root node would
    be useful, or perhaps the NTP node.

### Offline operation

System must function when offline without valid time. Again, for the point merge
algorithm to work correctly, timestamps for new points coming into the store
must be newer than what is currently stored. There are two possible scenarios:

- Problem: system powers up with old time, and points in DB have newer time.
  - Solution: if we don't have a valid NTP time, then set system time to something
    later than the newest point timestamp in the store.
- Problem: NTP sets the time "back" and there are newer points in the DB.
  - Solution: when we get a NTP time sync, verify it is not significantly earlier
    than the latest point timestamp in the system. If it is, update the point
    timestamps in the DB with the current time - 1yr. This ensures that settings
    upstream (which are likely newer than the edge device) will update the points 
    in the edge device.

We currently don't queue data when an edge device is offline. This is a different
concern which we will address later.

### Tracking the latest point timestamp

It may make sense to write the latest point timestamp to the store meta table.

### Syncing time from Modem or GPS

Will consider in future. Assume a valid network connection to NTP server for now.

## Reference/Research

### NTP

- https://wiki.archlinux.org/title/systemd-timesyncd
- `timedatectl status` produces following output:

```
               Local time: Thu 2023-06-01 18:22:23 EDT
           Universal time: Thu 2023-06-01 22:22:23 UTC
                 RTC time: Thu 2023-06-01 22:22:23
                Time zone: US/Eastern (EDT, -0400)
System clock synchronized: yes
              NTP service: active
          RTC in local TZ: no
```

There is a [systemd-timedated](https://www.freedesktop.org/software/systemd/man/org.freedesktop.timedate1.html) D-Bus API.

## Decision

what was decided.

objections/concerns

## Consequences

what is the impact, both negative and positive.

## Additional Notes/Reference
