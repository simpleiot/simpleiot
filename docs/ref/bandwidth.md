# Bandwidth

Many connected products reach the cloud over cellular links billed by the
megabyte. Low-cost IoT plans often carry 10 to 50 MB a month, which is about 1
MB a day, or 12 bytes a second averaged over the month. This page shows where a
Simple IoT device's traffic to its upstream comes from, what each part costs,
and which settings bring a device within that budget.

The figures below were measured on a device and an upstream connected over plain
NATS, with a proxy logging every protocol message between them. They count NATS
protocol bytes. The carrier also bills IP and TCP headers and TCP
acknowledgments, which add roughly 20% to 50% for messages this small, and TLS
adds a little per message on top. Treat the numbers as a lower bound and leave
headroom.

## Where the bytes go

A device that syncs with an upstream has a fixed cost to stay connected and a
cost for every point it writes.

| Source                                   | Per day    | Per month | Setting                              |
| ---------------------------------------- | ---------- | --------- | ------------------------------------ |
| Configuration stream keepalive           | 0.60 MB    | 18 MB     | fixed (see [below](#the-keepalive))  |
| Connection pings                         | 0.01 MB    | 0.2 MB    | fixed                                |
| Sync byte counters (hourly, the default) | 0.02 MB    | 0.7 MB    | `bytesReportPeriod` on the sync node |
| Sync byte counters (every minute)        | 1.3 MB     | 39 MB     |                                      |
| Store metrics (off by default)           | 2.8 MB     | 84 MB     | **Store metrics** on the device node |
| One point a minute                       | 0.65 MB    | 19 MB     | the client writing the point         |
| One point every 15 minutes               | 0.04 MB    | 1.3 MB    |                                      |
| One reconnect                            | about 8 KB |           | link quality                         |

An idle device with the defaults uses about 0.6 MB a day, or 19 MB a month,
nearly all of it the keepalive. A 30 MB plan then leaves about 11 MB a month for
data, which is roughly one point every two minutes across the whole device.

## What a point costs

Every point written anywhere in a device's tree is stored in the device's stream
and replicated upstream. Each one costs about 450 bytes on the link:

- **About 260 bytes to publish.** The subject carries the device ID twice and
  the node ID, `inst.<device>.<device>.<node>.p.<type>.<key>`, about 150
  characters, followed by a reply inbox named for the device key and the encoded
  point.
- **About 190 bytes for the upstream's acknowledgment**, which names the stream
  and the sequence number.

The value itself is a few bytes, so the cost tracks the number of points a
device writes. Writing fewer points is the most effective way to save data: most
clients have a period point on their node (`pollPeriod`, `period`, and the like)
that sets how often they sample.

## The keepalive

The device reads the configuration its upstream writes for it through a
JetStream pull consumer on the upstream's `inst_X_R` stream (see
[Synchronization](sync.md)). A pull request stays open on the upstream for 5
minutes and is then renewed, and the upstream sends an idle heartbeat every 30
seconds so the device can tell that the request is still alive. Together these
cost about 2 KB every 5 minutes.

30 seconds is the longest heartbeat interval the NATS client allows for a pull
consumer, so this is the floor for the current design. A push consumer would
need no heartbeat at all and could bring an idle device close to zero; that is
possible future work.

Connection pings are separate and small, a few bytes every couple of minutes.
They also keep the carrier's NAT mapping for the connection alive, which
prevents the extra cost of a reconnect.

## Connecting and reconnecting

Each time the link drops and comes back, the device and the upstream exchange
server information, credentials, and stream and consumer state before data flows
again: about 8 KB on a small tree. A device on a link that drops ten times a day
spends about 2.4 MB a month on reconnects alone, so a stable link matters.

The first connection also sends the device's entire stream, its whole tree and
history, from the beginning. A bare device sends about 14 KB. A device that
collected data before it was first connected sends all of it, so connect a
device once on a cheaper network, such as Wi-Fi or Ethernet during provisioning,
when it has history to deliver.

## Measuring a device

The sync node reports `txBytes` and `rxBytes`, the totals sent to and received
from the upstream, counted at the network socket so TLS overhead is included.
Each report is itself a pair of points, so a short `bytesReportPeriod` costs
data of its own; leave it at the hourly default on a metered link and shorten it
only while investigating. See [Synchronization](../user/sync.md#schema).

To find what a device is writing, run `siot log` against it and watch which
points change and how often, or list the per-subject message counts in its
stream with `nats stream subjects`. A subject with many more messages than the
rest is usually the place to start.

## Budget checklist

For a device on a plan of tens of megabytes a month:

- Leave **Store metrics** off, and `bytesReportPeriod` at its hourly default.
- Count the points the device writes. A point written once a minute costs about
  19 MB a month, so on a 30 MB plan the whole device can afford about one point
  every two minutes.
- Do the first sync of a device with history on an unmetered network.
- Watch `txBytes`, `rxBytes`, and the sync count for a few days after
  deployment. A rising sync count means reconnects, which cost data even when
  nothing changes.

## Ideas

None of these are implemented. They are directions for bringing the cost of a
point, and of an idle connection, down further. The estimates are rough and each
idea needs measuring before it is adopted.

### Shorter IDs

Node and instance IDs are UUIDs written as 36 characters. The same 128 bits
encoded as base64url take 22 characters, and base64url uses only characters that
are valid in NATS subject tokens and stream names. A point's subject carries
three IDs and the acknowledgment names a stream built from two, so shorter IDs
would save about 70 bytes per point. IDs are stored and shown to users, so this
changes every existing tree.

### Leave the boundary and origin out of the wire subject

Every publish repeats `inst.<device>.<device>.` because the upstream's replica
stream captures that subject. The upstream already knows which device a
credentialed connection belongs to, so the device could publish a short subject
such as `d.<node>.p.<type>.<key>` and the upstream could rewrite it into the
stored form. NATS offers two places to do that: a subject transform on the
replica stream (`SubjectTransform` in the stream configuration), or a subject
mapping on the device's account. The device's local stream keeps the full
subjects, so the pump would rewrite them as it sends.

### Short aliases for boundary and origin pairs

Where the full subject has to cross the link, the upstream could assign each
(boundary, origin) pair a short alias when the device enrolls, stored on its
credential node: `inst.<device>.<device>` becomes, say, `a.17`. The replica
stream would capture `a.17.>` and transform it to the stored form. The same idea
could extend to node IDs, with a per-device table of short node numbers, at the
cost of keeping that table consistent on both sides.

### Shorter reply inboxes

Each request and each acknowledgment carries a reply inbox that starts with the
device's 56-character public key. Any prefix the upstream can derive from the
key works for scoping replies, for example a 22-character hash, which would save
about 35 bytes in each direction per point.

### Batching

The pump publishes each point as its own message and waits for its own
acknowledgment, which is about 40% of a point's cost. NATS 2.12 and later can
acknowledge a batch as a whole: atomic batch publish (`AllowAtomicPublish`)
stores a batch all at once and acknowledges only its last message, and fast
batch publish (`AllowBatchPublish`) acknowledges periodically. Either fits the
pump's existing windows, and would remove most acknowledgments.

Holding points on the device and sending them in bursts, every few minutes for
example, would also put more data in each packet, so the IP and TCP headers the
carrier bills are shared across many points. Fewer, larger transmissions also
let a cellular radio stay idle longer, which saves power.

### Compression

Point messages repeat the same subjects over and over, which compresses well
when the compressor can remember earlier messages. NATS supports compression on
WebSocket connections (`Compression` in the client options, with compression
enabled on the server's WebSocket listener), and sync already accepts `ws://`
URIs.

A test build with compression turned on in the client, the server's WebSocket
listener, and the HTTP port's WebSocket proxy measured this, with a point
written every 5 seconds:

| Link                        | Total   |
| --------------------------- | ------- |
| `nats://`                   | 102 B/s |
| `ws://` without compression | 104 B/s |
| `ws://` with compression    | 84 B/s  |

WebSocket framing adds about 2%, and compression saves about 19%. The saving is
modest because the NATS client and the WebSocket library both compress each
message on its own, so a subject repeated on every point is never reused.

A compressor that keeps its state across messages does far better. Replaying a
recorded sync stream (a point every 5 seconds, plus the idle traffic) through
compressors that flush after every write, so no message waits:

| Compression                         | Bytes, both directions | Saving |
| ----------------------------------- | ---------------------- | ------ |
| None                                | 30,921                 |        |
| Deflate, each message on its own    | 24,654                 | 20%    |
| Deflate, state kept across messages | 3,601                  | 88%    |
| Same, with a 4 KB window            | 3,792                  | 88%    |
| zstd, state kept across messages    | 3,736                  | 88%    |

A point's publish shrinks from about 260 bytes to about 30, because the subject,
reply inbox, and stream name repeat almost exactly from one message to the next.
This is the largest single saving on this page, and it needs no change to
subjects, IDs, or how sync replicates streams.

Neither the NATS client nor the server offers this on a client connection, so
Simple IoT would provide the transport itself: a connection wrapper that
compresses each write with deflate or zstd and flushes it, used by the sync
client's dialer for a new URI scheme, and a listener on the upstream that
decompresses and hands the connection to the embedded NATS server. TLS sits
outside the compression, since compressed ciphertext saves nothing.

With messages this small, the IP and TCP headers on each packet (about 52 bytes,
plus acknowledgment packets) become most of what the carrier bills, so
compression pairs well with sending points in bursts.

### A push consumer for configuration

A push consumer delivers configuration from the upstream without the device
renewing a pull request, so it needs no idle heartbeat and would remove most of
the 18 MB a month an idle device spends on the keepalive. The pump's windows
rely on in-order redelivery, which would need to be reworked for push delivery.

### Faster reconnects

TLS session resumption would shorten the handshake on each reconnect, and the
device could skip requests whose answers it already holds, such as the upstream
root, when it reconnects to the same upstream.
