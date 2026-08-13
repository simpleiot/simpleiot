# Data Synchronization

Simple IoT synchronizes data between instances by **replicating the store's
streams** rather than by comparing and copying tree state. Each instance appends
only to its own streams (see [Store](store.md)); other instances hold replicas
of those streams and merge them at read time. This page explains how the pieces
of NATS — core subjects, JetStream streams, durable consumers, and message
headers — combine to do this.

[ADR-7](../adr/7-jetstream-store.md) records the design analysis. The previous
implementation, which compared Merkle-style node hashes and pushed subtrees, is
fully replaced; streams carry their own sequence numbers, so nothing needs to be
compared to know what is missing.

## The model in one paragraph

Every stream has exactly one writing instance (its _origin_). A device with root
`X` writes everything to its stream `inst_X_X`; a hub with root `R` writes
configuration for the device's subtree to its own stream `inst_X_R`. Sync means
each side keeps a copy of the other's stream: the hub holds a replica of
`inst_X_X`, the device holds a replica of `inst_X_R`. Current state on either
side is the merge of the subject tips of both streams — newest timestamp wins,
with a deterministic origin tie-break. Because no instance ever writes remote
data into its _own_ streams, points cannot echo back and forth between
instances: there is no loop to suppress.

```
   device X                                hub R
  ┌───────────────────┐                  ┌───────────────────┐
  │ inst_X_X (owned)  │ ──── push ─────► │ inst_X_X (replica)│
  │ inst_X_R (replica)│ ◄─── pull ────── │ inst_X_R (owned)  │
  │                   │                  │ inst_R_R (owned)  │
  └───────────────────┘                  └───────────────────┘
     merge tips of both                     merge tips of both
     = current state                        = current state
```

## How each NATS feature is used

**Core NATS subjects (`p.>`, `ep.>`, `up.>`)** carry points between components
_within_ an instance in real time, exactly as before — clients, rules, and the
UI are unaware of synchronization. The store subscribes to these wire subjects,
persists local writes to its origin streams, and fans points out to `up.>`
subjects for listeners like rules and database clients.

**JetStream streams** are both the store and the unit of sync. Because storage
subjects embed the boundary and origin (`inst.<boundary>.<origin>.…`), a replica
stream on another instance can use the same name and subjects — a copied message
needs no translation.

**Durable consumers** drive replication. The sync client (which runs on the
downstream instance and connects to the upstream's NATS server, using the URI
and auth token on its Sync node) runs two pumps:

- _push_: a durable consumer on the local `inst_X_X` delivers each message, and
  the pump publishes it — same subject, same payload — to the upstream, where
  the replica stream captures it.
- _pull_: a durable consumer on the upstream's `inst_X_R` does the same in the
  other direction.

A pump moves messages in windows of up to 256, so the round trips overlap rather
than running one at a time — which is what makes a first sync of a long stream
practical. A window is acknowledged only after the receiving side confirms every
message in it, and a window that fails is resent as a unit with none of it
acknowledged. That is what keeps each subject in source order: acknowledging the
part that landed would let the resend of a failed message arrive after messages
stored later, and the receiving store reads the last message on a subject as
that subject's current value.

A durable consumer remembers its position across disconnects, so a reconnect
delivers exactly the messages the other side missed — no rescan, no comparison.
This is what replaces the hash tree: the stream sequence _is_ the
synchronization state.

The durable is named for the receiving instance, so an instance that loses its
identity — a store reset gives it a new root ID — is a new reader as far as the
sender is concerned, and receives the sender's retained history from the
beginning.

The pumps move messages and nothing else: they create a missing replica stream
but never change an existing stream's configuration. Each instance's store owns
the configuration of the streams on its own disk and applies its retention
policy to replica streams when it discovers them, so a hub can keep more (or
less) history of a device's data than the device keeps itself.

**Message headers** solve origin attribution. When a store consumes a replica
stream, it merges each message into its caches and, when a tip changes,
re-broadcasts it on the ordinary wire subjects so local clients react — tagged
with a `Siot-Origin` header naming the writing instance. A store receiving a
wire message tagged with a remote origin merges it and fans it out but **never
persists it**; the replica stream is the persistent copy. This single rule keeps
the single-writer property intact everywhere.

## Life of a connection

1. The sync client connects to the upstream NATS server (plain NATS or NATS over
   WebSocket).
2. **Adoption:** if the upstream tree has no node with this instance's root ID,
   the client announces itself with one edge message; the upstream persists a
   device node under its root. (This is an ordinary untagged write — from the
   upstream's view it is its own edge, in its own boundary.)
3. The push pump ensures the replica stream exists upstream and starts copying;
   the device's whole tree — structure, configuration, and history — arrives
   through it, from sequence 1 on first connect.
4. The pull pump discovers upstream-origin streams for this instance's boundary
   and copies them down; the first hub-side configuration write creates
   `inst_X_R`, and the device picks it up on its next scan.
5. Each store's replica consumers merge the arriving messages and re-broadcast
   changed tips locally.

Configuration written on the hub _before_ the device ever connects
(pre-provisioning) simply waits in `inst_X_R` and arrives on first connect.

## Offline catch-up

While disconnected, both sides keep writing to their own streams. On reconnect,
the durable consumers resume and deliver only the backlog. Two kinds of
consumers see that backlog differently:

- **State clients** (rules, protocol clients, the UI) should not see a replay of
  stale intermediate values. The store therefore holds re-broadcasts while a
  replica consumer has a backlog and emits one message per changed subject — the
  final tip — once it drains.
- **History** needs every point. It is preserved automatically: the replica
  stream receives the full backlog in order with original embedded timestamps,
  so local history stays gap-free (up to each stream's retention limit). The Db
  (InfluxDB) client works this way: it consumes the streams with its own durable
  consumers, so an external time-series database receives every point —
  including the backlog after a sync outage or the client's own downtime —
  rather than only what happened to cross the wire while it was listening.
  External sinks can follow the same pattern.

## Conflicts

Concurrent writes to the same point from two instances are rare in practice — a
sensor value has one source, a setting is usually edited in one place. When they
happen, every instance applies the same merge rule to the same streams: newest
embedded timestamp wins, and equal timestamps resolve to the lexically greater
origin ID, so all instances converge on the same value without coordination.

## Deleting a device (detach)

The edge that attaches a device into the hub's tree lives in the hub's own
boundary stream, which the device does not replicate. Tombstoning that edge on
the hub therefore _detaches_ the device: the hub stops showing it, while the
device keeps operating standalone, unaware. The device does not force itself
back into the tree; only the hub can restore the edge (undelete), after which
replication resumes where it left off.

## Current limitations and direction

- Only the instance's root boundary replicates today; nested device boundaries
  (a device that itself syncs devices) are planned.
- Replication runs over the ordinary upstream client connection. JetStream
  _sourcing_ across NATS leaf connections — where the NATS servers replicate the
  streams themselves — is verified to work (see `store/leafnode_spike_test.go`)
  and is the intended replacement once per-instance JetStream domain
  configuration is worked out.
- Authorization currently uses the shared auth token; per-stream permissions
  issued via NATS auth callout are the planned tightening, and the
  stream-per-boundary layout is what makes one-rule-per-device grants possible.

See the
[Stage 3 plan](https://github.com/simpleiot/simpleiot/blob/master/plans/2026-08-06-stage3-jetstream-sync.md)
for the full status list.
