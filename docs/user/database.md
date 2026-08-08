# Database Client

The main [SIOT store](../ref/store.md) is NATS JetStream, which retains a
bounded history of points for each node. For long-term storage, dashboards, and
ad-hoc queries, a Database client can forward points to an external time-series
database.

[VictoriaMetrics](https://victoriametrics.com/) is the primary time-series
store for SIOT. The Database client speaks the InfluxDB v2 write API, which
VictoriaMetrics
[supports](https://docs.victoriametrics.com/#how-to-send-data-in-influxdb-v2-format),
so InfluxDB 2.x can also be used.

## Reliable delivery with durable consumers

The Database client reads points from the store's JetStream streams using
durable consumers rather than subscribing to live message traffic. A durable
consumer is a named position in a stream that the NATS server persists to disk
alongside the stream data. The client acknowledges each message after handing
its points to the database writer, and the server advances the saved position
only on acknowledgment.

This makes delivery resumable. If the Database client, the SIOT instance, or
the connection to the database is down for a period of time, points continue to
accumulate in the streams. When the client comes back, delivery resumes from
the saved position and the missed points are written to the database. Each
Database node keeps its own position, so multiple Database clients can consume
the same streams independently.

Two limits apply:

- Stream retention bounds how far behind a client can fall. By default the
  store keeps the last 5000 points per subject (one subject is one point type
  and key on one node). If a client is down long enough that a signal exceeds
  this limit, the oldest points for that signal are dropped from the stream and
  will be missing from the database.
- High-rate points are not stored in streams. They are delivered live and are
  not recovered after downtime.

A newly added Database node starts recording from the present; it does not
backfill history already in the streams. Streams that appear after the client
starts, such as the replica stream for a newly adopted device, are consumed
from their beginning so the device's initial catch-up is captured.

## Choosing a database type

Add a Database node and choose the database type: **InfluxDB 2.x** or
**Victoria Metrics**. Both are written using the InfluxDB version 2 line
protocol, so the connection settings are similar, but the two differ in what
they store and in how you graph the result. Existing Database nodes have no
database type set and continue to behave as InfluxDB.

## Victoria Metrics

Set the database type to Victoria Metrics and set the URI to the write
endpoint, typically `http://myserver:8428` for a single-node instance.
VictoriaMetrics has no concept of an organization or a bucket, so those fields
are hidden when this type is selected.

A single-node VictoriaMetrics has no authentication on the write path, so the
Auth Token field can be left blank. VictoriaMetrics expects authentication to
be handled by [vmauth](https://docs.victoriametrics.com/vmauth/) or vmgateway
in front of it. The client sends the token as an `Authorization: Token <token>`
header, which is one of the formats vmauth accepts, so setting the Auth Token
here works when you point the URI at vmauth.

VictoriaMetrics
[does not support storing strings](https://stackoverflow.com/questions/66406899/does-victoriametrics-have-some-way-to-store-string-value-instead-float64);
it
[converts any non-numeric field value to 0](https://docs.victoriametrics.com/victoriametrics/integrations/influxdb/).
The client therefore writes only the numeric `value` field and skips string
points, which keeps a `points_text` series of zeros out of the database. If you
want to filter or graph on a value, publish it as a number. The
[GPS client](gps.md) does this for its fix status points for exactly this
reason.

Each point arrives in VictoriaMetrics as the metric `points_value`, with the
point `type` and `key` and all of the node tags described below available as
labels.

### Graphing Victoria Metrics data

Use Grafana with a Victoria Metrics (Prometheus-compatible) data source and
query `points_value` with MetricsQL. See the
[Graphing documentation](graphing.md) for how the node tags map to graph labels.

## InfluxDB 2.x

Point data can also be stored in an InfluxDB 2.x database by adding a Database
node:

<img src="assets/image-20240319111031186.png" alt="image-20240319111031186" style="zoom:50%;" />

## Tags

The following tags are added to every point written to the database:

- `node.id` (typically an UUID)
- `node.type` (extracted from the type field in the edge data structure)
- `node.description` (generated from the `description` point from the node)

### Custom Tags

Additional tag points can be specified. The DB client will query and cache node
points of these types for any point flowing through the system and then add
tags in the format: `node.<point type>.<point key>`. In the below example, we
added a machine tag to the signal generator node generating the data.

<img src="assets/image-20240319112828216.png" alt="image-20240319112828216" style="zoom:50%;" />

When the `tag` field is specified in the database node, this `machine` tag is
now added to the tags for every sample.

- `value` and `type` and fields from the point
- `node.description` and `node.type` are automatically added
- `node.tag.machine` got added because the `tag` point was added to the list of
  node points that get added as tags.

![image-20240319110846431](assets/image-20240319110846431.png)

See the [Graphing documentation](graphing.md) for information on how to
automatically map tags to graph labels.

The database indexes tags, so generally there is not a huge cost to adding tags
to samples as the long string is only stored once.

### Tag inheritance

Tag points are inherited from ancestor nodes, so a label can be set once on the
node that represents the thing being described — a machine group, a device, a
site — and every point emitted beneath it carries that tag. For example, with
`tagPointType` set to `tag`:

```
device        tag: site=plant-a, customer=acme
└── press-3   tag: machine=press-3, site=plant-b
    └── temp-1    tag: sensor=inlet
```

a point emitted by `temp-1` is written with `node.tag.sensor=inlet`,
`node.tag.machine=press-3`, `node.tag.site=plant-b`, and
`node.tag.customer=acme`.

The resolution rules are:

- All tags resolve into the same flat `node.<point type>.<point key>`
  namespace, so queries do not depend on the depth at which a tag was set.
- When the same tag is defined at more than one level, the value closest to the
  emitting node wins, so a local tag overrides an inherited one (`site` above).
- Inheritance stops at the Database client's parent node, whose own tags are
  included. Nodes above the Database client's scope do not contribute tags.
- A node can have more than one parent. When two ancestors at the same depth
  define the same tag, the node with the lowest ID wins, and the client logs
  the ambiguity the first time it is seen.
- `node.id`, `node.type`, and `node.description` always describe the emitting
  node and are never inherited.

Tag changes are not retroactive: editing a tag starts a new series in the
database from that point forward, and queries spanning the change see both
series. The same applies to `node.description`.

## Schema

Below is an export of a Victoria Metrics node and an InfluxDB node:

```yaml
nodes:
  - db:
      dbType: victoriaMetrics
      description: Victoria Metrics
      tagPointType: tag
      uri: http://localhost:8428
  - db:
      authToken: T0k3n
      bucket: siot
      dbType: influxdb
      description: InfluxDB
      org: bec
      tagPointType:
        - machine
        - tag
      uri: http://localhost:8086
```

`dbType` is `victoriaMetrics` or `influxdb`; a node with no `dbType` behaves as
InfluxDB. `org` and `bucket` apply to InfluxDB alone, and Victoria Metrics
nodes leave them out.

`tagPointType` is a list, so a single custom tag is written as one value and
several are written as a sequence. Each entry is a point type, and the client
adds it to every sample as `node.<point type>.<point key>`.

An export carries `authToken` as it was entered, so treat a file that contains
database nodes the way you would treat the token itself.
