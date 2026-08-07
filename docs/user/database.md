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

## Victoria Metrics

To store points in VictoriaMetrics, add a Database node and set the URI to the
VictoriaMetrics server (default port 8428, for example
`http://localhost:8428`). The org, bucket, and auth token settings can be left
blank unless your installation requires them.

VictoriaMetrics
[does not support storing strings](https://stackoverflow.com/questions/66406899/does-victoriametrics-have-some-way-to-store-string-value-instead-float64).
Note what this means in practice. The database client writes each point with
both a `value` field and a `text` field. VictoriaMetrics splits those into
separate series named `points_value` and `points_text`, and
[converts any non-numeric field value to 0](https://docs.victoriametrics.com/victoriametrics/integrations/influxdb/).
The line is still accepted, so numeric points are stored correctly, but the
content of a text point is lost. If you are storing data in VictoriaMetrics and
want to filter or graph on a value, publish it as a number. The
[GPS client](gps.md) does this for its fix status points for exactly this
reason.

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
