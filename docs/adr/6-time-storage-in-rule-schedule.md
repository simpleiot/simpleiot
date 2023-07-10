# Time storage in rule schedules

- Author: Cliff Brake, last updated: 2023-07-10
- PR/Discussion:
- Status: discussion

## Problem

When storing times/dates in rule schedules, we store time as UTC, but this can
be problematic when there is a time change. In once application, SIOT plays a
chime at a certain time of day, but when time changes (daylight savings time),
we need to adjust the time in the rule and this is easy to forget.

## Context/Discussion

UTC was chosen as the storage format for the following reasons:

- it is universal -- it always means the same thing everywhere
- typically in UI or reports, times are translated to users local times
- server and edge devices can operate in UTC without needing to worry about
  local time
- rules run on cloud instances have a common timebase to work from. In a highly
  distributed system, you may have device in one timezone trigger an action in
  another time zone.

However, must applications (building automation, etc.) run in a single location,
and the loss or gain of an hour when the time changes is very inconvenient.

### Reference/Research

- [TMPDIR Forum discusion](https://community.tmpdir.org/t/daylight-savings-time-dst-in-iot-applications/1092)

## Decision

what was decided.

objections/concerns

## Consequences

what is the impact, both negative and positive.

## Additional Notes/Reference
