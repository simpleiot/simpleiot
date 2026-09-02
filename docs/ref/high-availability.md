# High Availability

Availability in Simple IoT is not one problem but two, and they have different
answers. **Keeping data safe when something is down** is largely solved by the
store and synchronization design. **Keeping the application serving continuously
through a failure** is not solved today, and this page describes what stands in
the way and which approaches fit the architecture.

Three facts from the current implementation shape every option below:

1. **Each stream has exactly one writing instance.** Only the origin instance
   appends to `inst_<boundary>_<origin>`. This single-writer property is what
   makes synchronization echo-free and the merge deterministic (see
   [Store](store.md) and [ADR-7](../adr/7-jetstream-store.md)).
2. **The store runs inside the SIOT process.** It subscribes to the core NATS
   wire subjects `p.>` and `ep.*.*` and turns what arrives into JetStream
   appends (`store/store.go`). NATS on its own stores nothing on those subjects.
3. **Streams are created with a single replica.** `CreateOrUpdateStream` in
   `store/jetstream.go` sets no `Replicas` field, so streams and the `META` KV
   bucket default to one. Nothing in the code asks for quorum.

## What the design already provides

The strongest availability property in the system is at the edge. A synced
device writes to its own local stream and replicates through a durable consumer,
so an upstream instance can be down for hours and the device loses nothing: on
reconnect the consumer resumes at its recorded position and delivers the
backlog, with original timestamps, in order. History stays gap-free up to the
stream's retention limit. See [Synchronization](sync.md) for the mechanism.

The same pattern protects consumers of the data. The Db client reads the
boundary-origin streams with its own durable consumers (`client/db.go`), so an
external time-series database receives every point across the client's own
downtime, not only what happened to cross the wire while it was listening.
External sinks can follow the same pattern.

What neither of these covers is a point published directly to a wire subject
while the application is stopped. That case is described in
[When SIOT is not running](#when-siot-is-not-running) below.

## Approaches to application redundancy

### Active/passive against a clustered NATS server

This is the approach that fits the architecture. NATS runs as a cluster,
JetStream streams are configured with three replicas, and several SIOT processes
contend for a lease so that exactly one is active at a time. The single-writer
property is preserved because only the active process writes.

JetStream supplies the election primitive directly: a revision-checked `Update`
on a KV key is a compare-and-set, so a leader lease can be built with no
dependency beyond what the store already uses.

Two pieces of work are prerequisites:

- **Replica count must be configurable.** A single-replica stream on a cluster
  lives on one server, so it is unavailable while that server restarts and it
  gains nothing from the cluster. Both `StreamConfig.Replicas` and the `META` KV
  bucket need to follow a setting.
- **Failover time needs measuring.** `loadAllStreams` reads every stream's
  subject tips into the caches before the store serves anything, so a standby's
  time to become useful scales with the number of subjects rather than the depth
  of history. This is worth measuring against a realistic tree before promising
  a recovery time.

### Active/active

Running two SIOT processes against one store does not work today, and the
reasons are structural rather than incidental. Both processes read the same root
ID from the shared `META` bucket, so both become the origin for the same
streams. Both subscribe to `p.>` and persist every point, which duplicates
appends (harmless to the merge, which is idempotent, but doubling storage), and
both answer `nodes.*.*` requests, so a requester receives two replies. Every
client also runs twice: rules fire twice, Modbus polls twice, database writes
happen twice.

Making this work would require per-process instance identity and arbitration
over which process owns which clients. That is a substantial change to the
model, and active/passive delivers most of the benefit without it.

### Peer instances replicating through synchronization

Two instances, each with its own root node and its own streams, replicating to
each other preserves the single-writer property cleanly. It is the natural
extension of what synchronization already does.

Two limitations apply today. Synchronization is shaped for a device-to-hub
relationship, and only the instance's root boundary replicates; nested device
boundaries are planned but not implemented. Failover also means clients
repointing at the surviving instance, since there is no shared address. This is
a resilience arrangement rather than a load-balancing one.

## Running against an external NATS server

SIOT can already run against a NATS server it does not start.
`--natsDisableServer` suppresses the embedded server, and `--natsServer` (or
`SIOT_NATS_SERVER`) points the process at another one. That covers self-hosted
clusters and hosted services such as Synadia Cloud as far as the connection
itself is concerned.

Four things need attention before a hosted cluster is a working deployment:

| Area               | Current state                                                                                                  | What is needed                                                                                                                                                  |
| ------------------ | -------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Authentication     | Token only (`nats.Token` in `server/server.go`)                                                                | An `nats.UserCredentials` option, since hosted services authenticate with NKey/JWT credentials                                                                  |
| Stream replicas    | One replica, not configurable                                                                                  | A replica setting applied to streams and the `META` bucket                                                                                                      |
| Frontend WebSocket | The UI connects over WebSocket through the HTTP server, presenting the user's JWT to the in-process authorizer | A hosted WebSocket endpoint that accepts the user JWT, which realistically means NATS auth callout against the same check                                       |
| Write latency      | `nodePoints` publishes each point synchronously and waits for its acknowledgment (`store/jetstream.go`)        | Measurement against the target cluster; a wide-area round trip with quorum replaces a sub-millisecond local write with tens of milliseconds, serially per point |

The frontend path is the largest piece of real work, and write latency is the
one most likely to constrain a hub with many devices. Both are worth proving
with a small deployment before committing to a hosted provider.

An external NATS server also moves the data. Backups, restores, and
`--resetStore` all act on storage that is no longer on the SIOT host, which
changes how an instance is operated and recovered.

A hosted cluster makes NATS highly available. The application is a separate
question, because the application is what persists points and runs clients. The
next section describes what that distinction costs when the application stops.

## When SIOT is not running

If the application is stopped and a point arrives on a wire subject, **the point
is discarded and the publisher is not told**.

The wire subjects `p.>` and `ep.*.*` are plain core NATS, which is at-most-once:
with no subscriber, the server drops the message and the publish still succeeds.
The store is the only thing that captures those subjects into a stream, and it
runs inside the process that is not running.

A hosted NATS server makes this quieter rather than better. The server stays up,
so publishes continue to succeed and there is no connection error to alert on.

| Source                       | Outcome while SIOT is stopped                                                             |
| ---------------------------- | ----------------------------------------------------------------------------------------- |
| External publishers to `p.>` | Discarded, no error returned                                                              |
| HTTP API                     | Unavailable; the API is served by the same process                                        |
| Synced downstream instances  | Safe: buffered in their own streams and delivered when the durable consumer resumes       |
| In-process clients           | Not running either, so the loss is the polling gap                                        |
| Db client                    | Its own downtime is covered by durable consumers, but only for points already in a stream |

There are two ways to close the gap for direct publishers.

The approach consistent with the design is to make the publisher a SIOT
instance, or to use the edge client, so its points land in a local stream first
and synchronization delivers the backlog. This is what synchronization exists to
do, and it needs no new mechanism.

The alternative is an ingest stream on the NATS server capturing `p.>` and
`ep.>`, which the store drains at startup with a durable consumer. That moves
durability from the application to the server, so points survive a restart of
the application. It is arguably a prerequisite for treating a hosted NATS
deployment as safe. The costs are a second write on every point in addition to
the storage-subject write, and design work on ordering and deduplication.
High-rate points on `phrup.>` would stay outside it, since they are deliberately
not stored.

## Where to start

The most valuable work is making sure data is buffered somewhere durable before
the application touches it. That is the gap that loses data rather than merely
pausing service, and clustering NATS does nothing for it. Synced devices already
have this property; direct publishers do not.

If continuous cloud service is the goal after that, active/passive against a
three-replica cluster with a KV-based lease is the shape that fits. The frontend
WebSocket path and per-point write latency are the two questions to answer
before selecting a hosted NATS provider.
