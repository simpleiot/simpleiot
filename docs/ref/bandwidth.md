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

The pump reads with the NATS client's `Messages`, which caps the heartbeat at 30
seconds. `Fetch` has no such cap: its heartbeat only has to be shorter than half
the request's expiry. A pump that fetched with a 30-minute expiry and a
15-minute heartbeat would send three messages every 30 minutes instead of twelve
every 5, bringing the keepalive from 18 MB to about 1 MB a month and sending
about one twentieth of the packets. A lost pull request would then take up to 30
minutes to notice; a dropped connection is still caught within minutes by the
connection pings. A push consumer would need no heartbeat at all; that is
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

### Send on change

The fewest bytes come from points that are never written. The GPIO, IIO, Modbus,
and 1-Wire clients already publish a value only when it changes (IIO with a
`minChange` deadband), and republish it every 10 minutes even when it does not,
so a graph and an upstream always have a recent sample. That refresh is most of
what an unchanging input costs.

The filtering belongs in the client that reads the input, which knows the
signal's units and noise, so local rules, local history, and the upstream all
see the same points and sync stays a faithful copy of the device's stream. A
shared helper for the IO clients could offer:

- a deadband for every analog input, as an absolute change or a percent of
  range, compared against the last published value so slow drift is still
  reported;
- a refresh of 15 minutes for unchanged values;
- a minimum interval, so a noisy input cannot flood the link;
- a debounce for discrete inputs, so a chattering contact sends one change.

The cost of one unchanging input, per month:

| Refresh    | Points a day | Today  | With 88% compression |
| ---------- | ------------ | ------ | -------------------- |
| 10 minutes | 144          | 1.9 MB | 0.23 MB              |
| 15 minutes | 96           | 1.3 MB | 0.16 MB              |

Ten unchanging inputs on a 15-minute refresh cost about 13 MB a month today, and
about 1.6 MB with the compression described [below](#compression). Compression
would bring the keepalive from 18 MB to about 2 MB a month as well, which leaves
most of a 30 MB plan for changes. At that size the IP and TCP headers on each
packet, which compression does not reach, become the larger cost, so sending
points in bursts matters more.

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

With the 88% [compression](#compression) below, a point's publish is about 31
bytes and an acknowledgment about 23. Sent on its own, a point also takes about
three packets (52 bytes of IP and TCP header each, once TCP acknowledgments are
counted) and two TLS records (about 22 bytes each), so the packaging outweighs
the point. A burst shares that packaging, and a single acknowledgment, across
every point in it. For a device writing one point a minute:

| How points are sent          | Per point | Per month |
| ---------------------------- | --------- | --------- |
| One at a time, uncompressed  | 650 B     | 28 MB     |
| One at a time, compressed    | 254 B     | 11 MB     |
| Every 5 minutes, compressed  | 76 B      | 3.3 MB    |
| Every 15 minutes, compressed | 46 B      | 2.0 MB    |

The cost is latency upstream: a change reaches the upstream up to one burst
interval late. Rules on the device still act at once, since they read the local
store, and a change that should not wait, such as an alarm, could send its burst
immediately.

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

Leaf node connections support S2 compression, but the NATS server compresses
each flush on its own, starting a new stream every time. A point a minute is a
flush a minute, so a leaf node saves about what per-message compression does.

With messages this small, the IP and TCP headers on each packet (about 52 bytes,
plus acknowledgment packets) become most of what the carrier bills, so
compression pairs well with sending points in bursts.

### A push consumer for configuration

A push consumer delivers configuration from the upstream without the device
renewing a pull request, so it needs no idle heartbeat and would remove most of
the 18 MB a month an idle device spends on the keepalive. The pump's windows
rely on in-order redelivery, which would need to be reworked for push delivery.
Fetching with a long heartbeat (see [The keepalive](#the-keepalive)) gets most
of the same saving with no change to how the pump orders deliveries.

### Faster reconnects

TLS session resumption would shorten the handshake on each reconnect, and the
device could skip requests whose answers it already holds, such as the upstream
root, when it reconnects to the same upstream.

### A 10 MB a month budget

A hypothetical device shows how these ideas add up. It has 10 analog and 5
discrete inputs, sends on change with a 15-minute refresh, and syncs over TLS
with the compression described [above](#compression). Each analog input changes
significantly 24 times a day and each discrete input 20 times, on top of its
refresh. The link drops twice a day.

The first column adds only send on change and compression. The second adds the
rest of the practical options on this page: batching every 5 minutes, a `Fetch`
keepalive with a 30-minute expiry and a 15-minute heartbeat (see
[The keepalive](#the-keepalive)), and connection pings every 5 minutes.

| Use                                       | Send on change and compression | All practical options    |
| ----------------------------------------- | ------------------------------ | ------------------------ |
| Refresh of unchanged inputs (1,440 a day) | 2.3 MB                         | 1.3 MB                   |
| Changes (340 a day)                       | 0.6 MB                         | 0.3 MB                   |
| Sync byte counters (hourly)               | 0.1 MB                         | 0.04 MB                  |
| Configuration keepalive                   | 2.2 MB                         | 0.1 MB                   |
| Configuration changes from the upstream   | under 0.1 MB                   | under 0.1 MB             |
| Connection pings                          | 0.2 MB (every 2 minutes)       | 0.1 MB (every 5 minutes) |
| Reconnects, with a full TLS handshake     | 0.4 MB                         | 0.4 MB                   |
| TLS record overhead                       | 5.7 MB                         | 0.9 MB                   |
| IP and TCP headers                        | 22 MB                          | 3.5 MB                   |
| **Total**                                 | **about 34 MB**                | **about 7 MB**           |

The estimates assume:

- a point compresses to 54 bytes, or 31 bytes when a batch shares one
  acknowledgment;
- each packet carries 52 bytes of IP and TCP header, and each message sent on
  its own takes about three packets once TCP acknowledgments are counted;
- each TLS 1.3 record adds about 22 bytes, and a full handshake about 5 KB;
- batching sends whatever has changed every 5 minutes, in about three packets
  per burst;
- the `Fetch` keepalive sends a pull request, receives one heartbeat, and
  receives one expiry notice every 30 minutes, about 80 bytes compressed.

Shorter IDs, subject aliases, and shorter reply inboxes are left out. With
compression keeping its state across messages, the repeated IDs they remove
already cost only a few bytes per point.

Compression shrinks what each message carries, but every message still costs a
packet, its headers, a TLS record, and an acknowledgment. Once messages are tens
of bytes, those per-packet costs are most of the bill: the configuration
keepalive's heartbeat every 30 seconds costs about 9 MB a month in headers
alone. Send on change and compression get the payload under budget; reaching 10
MB also takes fewer packets, which is what batching, a long `Fetch` heartbeat,
and less frequent pings provide.

The ping interval is the remaining fixed cost. Pings keep the carrier's NAT
mapping alive, and carriers differ in how long an idle TCP connection survives,
so measure on the target network before stretching it.
