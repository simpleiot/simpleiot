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
  - synchronization may be a problem here if NTP client sends messages before a 
    client has started.
1. query for system state, and NTP sync status could be a field in this state.
  - should this be part of the root device node?
  - or a special hard-coded message?
  - it would be useful to track system state as a standard point so it gets
    synchronized and stored in influxdb, therefore as part of the root node would
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
    timestamps in the DB that are newer than the current time with the current time - 1yr. 
    This ensures that settings
    upstream (which are likely newer than the edge device) will update the points 
    in the edge device. This is not perfect, but if probably adequate for most systems.

We currently don't queue data when an edge device is offline. This is a different
concern which we will address later.

The SIOT synchronization and point merge algorithm are designed to be simple
and bandwidth efficient (works over Cat-M/NBIOT modems). There are design trade-offs. 
It is not a full-blown
replicated, log-based database that will work correctly in every situation. It is designed
so that changes can be made in multiple locations while disconnected and when
a connection is resumed, that data is merged intelligently. Typically, configuration
changes are made at the portal, and sensor data is generated at the edge, so this
works well in practice. 
When in doubt,
we prioritize changes made on the upstream (typically cloud instance), as that
is the most user accessible system and is where most configuration changes will be 
made. Sensor data is updated periodically, so that will automatically get refreshed
typically within 15m max. The system works best when we have a valid time at every 
location so we advise ensuring reliable network connections for every device, and at 
a minimum have a reliable battery backed RTC in every device.

### Tracking the latest point timestamp

It may make sense to write the latest point timestamp to the store meta table.

### Syncing time from Modem or GPS

Will consider in future. Assume a valid network connection to NTP server for now.

### Tracking events where time is not correct

It would be very useful to track events at edge devices where time is not
correct and it requires a big jump to be corrected. 

TODO: how can we determine this? From systemd-timedated logs?

This information could be used to diagnose when a RTC battery needs replaced, etc.

### Verify time matches between synchronized instances

A final check that may be useful is to verify time between synchronized instances are 
relatively close. This is a final check to ensure the sync algorithm does not wreak havoc
between systems, even if NTP is lying.

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
