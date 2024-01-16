# Jetstream SIOT Store

- Author: Cliff Brake, last updated: 2024-01-15
- PR/Discussion:
- Status: discussion

## Problem

SQLite has worked well as a store but the current store has a few problems:

- history is not synchronized
  - if a devices or server is offline, only the latest state is transferred when
    connected
- we have to re-compute hashes all the way to the root node anytime something
  changes
  - this does not scale to larger systems
  - is difficult to get right if things are changing while we re-compute hashes
  - a correct solution would probably require more locks making it scale even
    worse

## Context/Discussion

background, facts surrounding this discussion.

NATS Jetstream is a stream based store where every message in a stream is given
sequence number. Synchronization is simple in that if a sequence number does not
exist on a remote system, the missing samples are sent.

### Reference/Research

links to reference material that may be

## Decision

what was decided.

objections/concerns

## Consequences

what is the impact, both negative and positive.

## Additional Notes/Reference
