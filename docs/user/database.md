# Database Client

The main [SIOT store](../ref/store.md) is SQLite. SIOT supports additional
database clients for purposes such as storing time-series data.

Add a Database node and choose the database type: **InfluxDB 2.x** or
**Victoria Metrics**. Both are written using the InfluxDB version 2 line
protocol, so the connection settings are similar, but the two differ in what
they store and in how you graph the result. Existing Database nodes have no
database type set and continue to behave as InfluxDB.

## InfluxDB 2.x

Point data can be stored in an InfluxDB 2.0 Database by adding a Database node:

<img src="assets/image-20240319111031186.png" alt="image-20240319111031186" style="zoom:50%;" />

The following InfluxDB tags are added to every point:

- `node.id` (typically an UUID)
- `node.type` (extracted from the type field in the edge data structure)
- `node.description` (generated from the `description` point from the node)

### Custom InfluxDB Tags

Additional tag tag points can be specified. The DB client will query and cache
node points of these types for any point flowing through the system and then
InfluxDB tags in the format: `node.<point type>.<point key>`. In the below
example, we added a machine tag to the signal generator node generating the
data.

<img src="assets/image-20240319112828216.png" alt="image-20240319112828216" style="zoom:50%;" />

When the `tag` field is specified in the database node, this `machine` tag is
now added to the Influx tags for every sample.

- `value` and `type` and fields from the point
- `node.description` and `node.type` are automatically added
- `node.tag.machine` got added because the `tag` point was added to the list of
  node points that get added as tags.

![image-20240319110846431](assets/image-20240319110846431.png)

See the [Graphing documentation](graphing.md) for information on how to
automatically map tags to graph labels.

InfluxDB indexes tags, so generally there is not a huge cost to adding tags to
samples as the long string is only stored once.

## Victoria Metrics

Victoria Metrics
[supports the InfluxDB version 2](https://docs.victoriametrics.com/#how-to-send-data-in-influxdb-v2-format)
line protocol, so the same client writes to it. Set the database type to
Victoria Metrics and point the URL at the write endpoint, typically
`http://myserver:8428` for a single-node instance. Victoria Metrics has no
concept of an organization or a bucket, so those fields are hidden when this
type is selected.

A single-node Victoria Metrics has no authentication on the write path, so the
Auth Token field can be left blank. Victoria Metrics expects authentication to
be handled by [vmauth](https://docs.victoriametrics.com/vmauth/) or vmgateway
in front of it. The client sends the token as an `Authorization: Token <token>`
header, which is one of the formats vmauth accepts, so setting the Auth Token
here works when you point the URL at vmauth.

Victoria Metrics
[does not support storing strings](https://stackoverflow.com/questions/66406899/does-victoriametrics-have-some-way-to-store-string-value-instead-float64);
it
[converts any non-numeric field value to 0](https://docs.victoriametrics.com/victoriametrics/integrations/influxdb/).
The client therefore writes only the numeric `value` field and skips string
points, which keeps a `points_text` series of zeros out of the database. If you
want to filter or graph on a value, publish it as a number. The
[GPS client](gps.md) does this for its fix status points for exactly this
reason.

Each point arrives in Victoria Metrics as the metric `points_value`, with the
point `type` and `key` and all of the node tags described above available as
labels.

### Graphing Victoria Metrics data

Use Grafana with a Victoria Metrics (Prometheus-compatible) data source and
query `points_value` with MetricsQL. See the
[Graphing documentation](graphing.md) for how the node tags map to graph labels.
