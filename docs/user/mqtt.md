# MQTT

Simple IoT can serve MQTT itself and turn published messages into points. See
the [PLC page](plc.md) for how MQTT compares with the other ways to bring plant
data in, and [ADR-8](../adr/8-iot-data-models.md) for how common MQTT payload
formats compare with the Simple IoT point model. The design is open for
discussion on the
[community forum](https://community.tmpdir.org/c/simple-iot/5).

Support comes in four pieces, each usable on its own:

1. **A built-in broker.** Gateways and sensors publish directly to Simple IoT,
   with no separate broker to deploy, secure, and update.
2. **Subscriptions.** An `mqtt` node and its `mqttSub` children map named topics
   into points.
3. **A topic schema.** Declare what your topic levels mean, and nodes are
   created automatically as data arrives.
4. **Sparkplug B.** Birth certificates describe the data, so the node structure
   builds itself. Covered in its own section below.

## The built-in broker

Simple IoT embeds a NATS server, and NATS includes an MQTT server. It needs
JetStream, which Simple IoT already runs, so serving MQTT is a port setting
rather than a new dependency:

```
SIOT_NATS_MQTT_PORT=1883
```

The port is disabled by default. Points worth knowing:

- The broker implements **MQTT 3.1.1**. Clients that require MQTT 5 are refused,
  which matters mostly for newer gateways that default to version 5.
- QoS 0, 1, and 2 are supported. Sessions and retained messages are stored in
  JetStream, so they survive a restart.
- When an auth token is configured (`SIOT_AUTH_TOKEN`), MQTT clients supply it
  in the password field of the connect packet, with any non-empty user name
  alongside it, which MQTT 3.1.1 requires whenever a password is present. Use
  TLS (`SIOT_NATS_TLS_CERT`/`SIOT_NATS_TLS_KEY`, which also serve the MQTT
  listener) whenever a connection leaves a trusted network.
- Published messages become NATS subjects, so anything connected to Simple IoT
  over NATS sees them. Topic levels convert as `/` to `.`, and a literal `.` in
  a topic converts to `//`. A Sparkplug topic of
  `spBv1.0/plant/DDATA/line3/tank` arrives on the NATS subject
  `spBv1//0.plant.DDATA.line3.tank`.

Try it with the Mosquitto command line tools:

```
mosquitto_sub -h localhost -p 1883 -t 'plant/#' -v
mosquitto_pub -h localhost -p 1883 -t plant/line3/tank -m '{"value":42.1}'
```

Add `-u siot -P $SIOT_AUTH_TOKEN` to both when a token is configured. The
[command line walkthrough](#trying-a-topic-schema-from-the-command-line) below
takes this further and creates nodes from published messages.

An external broker still makes sense when the plant already runs one, when you
bridge several sites, or when you need broker features such as clustering or
fine-grained access control. Connecting to an external broker as a client is
planned as well, so the choice stays open.

## Subscriptions

An `mqtt` node holds the connection, and each `mqttSub` child maps one topic
into points:

```yaml
nodes:
  - mqtt:
      description: Plant data
      uri: "" # blank uses the built-in broker
      children:
        - mqttSub:
            description: Tank level
            topic: plant/line3/tank/level
            path: $.value
            units: cm
```

A blank `uri` uses the broker built into this instance, which is the only mode
available today -- setting a `uri` reports an error on the node until external
brokers are supported. `mqttSub` settings:

| Setting    | Purpose                                                    |
| ---------- | ---------------------------------------------------------- |
| `topic`    | The topic to subscribe to                                  |
| `path`     | Where in a JSON payload the value lives, such as `$.value` |
| `units`    | Engineering units, carried on the emitted points           |
| `scale`    | Multiplier applied to numeric values, 1 when unset         |
| `offset`   | Added after scaling: `value = raw * scale + offset`        |
| `disabled` | Stops the subscription without deleting the configuration  |

Each subscription also carries a tag point named `topic` holding the full topic,
so a series can be traced back to the message that produced it. See the
[graphing section of the PLC page](plc.md#graphing-plc-data).

A payload that does not parse, or a path that is not in it, sets an `error`
point on the subscription node and leaves the rest running.

### Payloads

Payloads are JSON, which covers the AWS IoT, Azure IoT, and gateway-defined
formats most installations use. How a payload maps depends on `path`:

- **`path` set:** the value at that location becomes a single point. Numbers
  become `value` points, strings become text points, and `true` and `false`
  become 1 and 0 so a rule compares them the way it compares any other on/off
  value.
- **`path` blank, payload is a bare number or string:** the payload itself
  becomes the point.
- **`path` blank, payload is an object:** each top-level field becomes a point,
  with the field name as the point key. A payload with twenty fields becomes one
  node and twenty points rather than twenty nodes, and the field name is
  queryable in the database as the `key` label.

A path is written in the dot notation JSON documentation generally uses:
`$.value`, `$.a.b`, and `$.a[0]` all work, and the leading `$` is optional.

Topics you have not named are ignored. A wildcard topic on one `mqttSub`
subscribes fine, but every match lands on that one node, so name topics
individually when they represent different things. The topic schema below is the
better tool when you want one rule to cover many topics.

## Automatic nodes with a topic schema

Plain MQTT carries no information about which topic level is a site and which is
a device, which is why nothing is created automatically by default. A topic
schema supplies that missing information. Declare what the levels mean on the
`mqtt` node, and matching topics create nodes as data arrives:

```yaml
nodes:
  - mqtt:
      description: Plant data
      uri: ""
      topicSchema: "{site}/{gateway}/{device}"
```

The first message on `plant-07/kepware-l3/press/tank_level` carrying
`{"value": 42.1}` creates:

```
Plant data (mqtt)
└── plant-07 (group, tag: site=plant-07)
    └── kepware-l3 (group, tag: gateway=kepware-l3)
        └── press (mqttDevice, tag: device=press)
              point: value, key tank_level
```

Expanding the `mqttDevice` node in the web UI lists every point it holds
alongside its current value, so you can see what a device is publishing without
querying the store.

The rules:

- **Each named level becomes a node**, carrying a tag named by its schema label.
  Intermediate levels are group nodes; the last named level is an `mqttDevice`
  node that receives the points. Levels written without braces are literals a
  topic has to match, so `plant/{site}/{device}` covers one prefix only.
- **Everything beyond the named levels becomes the point key.** Remaining topic
  levels and JSON field names join into the key with `/`, so a deeper topic
  extends the key rather than the node tree. A payload carrying a single field
  named `value` is treated as a scalar, since that is the shape a gateway
  publishing one measurement per topic uses.
- **Nodes are matched by topic identity, not by name.** Renaming a description
  or adding tags to an auto-created node survives later messages and restarts,
  and nothing is duplicated.
- **Nodes are never deleted automatically.** A quiet sensor and a removed sensor
  look the same from outside, so removal stays a human decision.
- **A `maxNodes` limit** (default 1000) guards against topics that carry
  unbounded values such as message IDs. When the limit is reached, an error
  point is set on the `mqtt` node and new topics are dropped.
- **Explicit `mqttSub` children win.** A topic named by a subscription is
  handled by that subscription alone, so hand-tuned mappings with units and
  scaling override the schema where precision matters.

The schema and explicit subscriptions compose well: start with a schema to see
what a site publishes, then add `mqttSub` entries for the values that need
units, scaling, or careful naming.

## Trying a topic schema from the command line

A topic schema is the quickest way to watch MQTT data turn into nodes, and the
Mosquitto command line tools are enough to exercise the whole path. Install them
with `apt install mosquitto-clients`, `pacman -S mosquitto`, or
`brew install mosquitto`, then start an instance with the broker enabled:

```
SIOT_NATS_MQTT_PORT=1883 siot serve
```

Add an `mqtt` node with a schema. Setting `debug: 1` logs every message the node
handles, which is worth having while testing:

```
cat <<EOF | siot import
apiVersion: 1
nodes:
  - mqtt:
      description: Plant data
      uri: ""
      topicSchema: "{site}/{gateway}/{device}"
      debug: 1
EOF
```

### One measurement per topic

Publishing a single value per topic is what most gateways do. The schema names
three levels, so the first message creates the site, gateway, and device nodes,
and the levels past the third become the point key:

```
mosquitto_pub -h localhost -p 1883 \
  -t plant-07/kepware-l3/press/tank_level -m '{"value":42.1}'
mosquitto_pub -h localhost -p 1883 \
  -t plant-07/kepware-l3/press/pump_rpm -m '{"value":1800}'
```

A payload that is a bare number or string works the same way, so a gateway that
publishes `1800` with no JSON around it needs nothing extra:

```
mosquitto_pub -h localhost -p 1883 -t plant-07/kepware-l3/pump/rpm -m 1800
```

### Several measurements in one payload

A gateway that publishes an object at the device level, the last level the
schema names, gets one point per field, with the field name as the point key:

```
mosquitto_pub -h localhost -p 1883 -t plant-07/kepware-l3/hmi \
  -m '{"line_speed":12.5,"state":"running","running":true}'
```

Numbers become value points, strings become text points, and `true` and `false`
become 1 and 0, so the `hmi` device ends up with `line_speed`, `state`, and
`running`.

Topic levels past the schema and field names inside the payload join into the
key with `/`, which means a deeper topic extends the key rather than the tree:

```
mosquitto_pub -h localhost -p 1883 \
  -t plant-07/kepware-l3/pump/motor/temp -m '{"value":38.5,"units":"C"}'
```

That message lands on the `pump` device as `motor/temp/value` and
`motor/temp/units`. An object holding one field named `value` is the scalar
case, so the same topic carrying `{"value":38.5}` produces the single key
`motor/temp`.

### Seeing what arrived

`siot export` prints the tree the messages built, tags included:

```
$ siot export
apiVersion: 1
nodes:
  - mqtt:
      description: Plant data
      topicSchema: "{site}/{gateway}/{device}"
      children:
        - group:
            description: plant-07
            id: plant-07
            tag:
              site: plant-07
            children:
              - group:
                  description: kepware-l3
                  id: kepware-l3
                  tag:
                    gateway: kepware-l3
                  children:
                    - mqttDevice:
                        description: press
                        id: press
                        tag:
                          device: press
                        value:
                          pump_rpm: 1800
                          tank_level: 42.1
```

`siot log` prints points as they arrive, which answers whether a value is
updating without reloading a page:

```
$ siot log
2026/08/20 14:32:09 NODE: hmi (mqttDevice) (799dd5f3-aa51-4245-a31d-5f3139cca804)
   - POINT: T:value V:12.500 K:line_speed O:d931bb99-e924-4fc2-86e7-d949ec942f2c 2026-08-20T14:32:09-04:00
```

`mosquitto_sub` shows the messages themselves, which separates a gateway that is
not publishing from a schema that is not matching:

```
mosquitto_sub -h localhost -p 1883 -t 'plant-07/#' -v
```

The same nodes appear in the web UI at `http://localhost:8118`, where you can
rename a device or add tags to it. Those edits survive later messages and
restarts, since auto-created nodes are matched by their `id` point rather than
by description.

### If nothing appears

- **Match the depth.** A schema of `{site}/{gateway}/{device}` needs three
  levels, so a message on `plant-07/kepware-l3` is ignored and logs nothing,
  even with `debug: 1` set.
- **Supply the token.** When `SIOT_AUTH_TOKEN` is set, a client that connects
  without it is refused with return code 5. Add `-u siot -P $SIOT_AUTH_TOKEN` to
  the `mosquitto_pub` and `mosquitto_sub` commands; the user name can be
  anything non-empty, which MQTT 3.1.1 requires alongside a password.
- **Stay on MQTT 3.1.1.** The Mosquitto clients use it by default, so no flag is
  needed. Passing `-V mqttv5` is refused by the broker.
- **Check the `mqtt` node for an error point.** A topic level carrying an
  unbounded value can reach `maxNodes` (1000 by default), after which new topics
  are dropped and the error point says so.
- **Watch for typos becoming nodes.** Nodes are never deleted automatically, so
  a mistyped topic leaves a node behind. Delete it in the UI once you are done.

## Sparkplug B

[Sparkplug B](https://sparkplug.eclipse.org/) adds a defined topic namespace, a
protobuf payload, and birth and death certificates on top of MQTT. Because an
edge node announces every metric it will report, with names and types, Simple
IoT builds the node structure from the data itself and no schema or subscription
list is required. Enable it on the `mqtt` node:

```yaml
nodes:
  - mqtt:
      description: Plant 03 Sparkplug
      uri: ""
      sparkplug: true
```

The topic namespace is `spBv1.0/{group}/{message type}/{edge node}/{device}`,
and it maps onto the graph directly:

```
Plant 03 Sparkplug (mqtt)
└── plant-03 (sparkplugGroup)
    └── ignition-edge (sparkplugNode)
        ├── press-1 (sparkplugDevice)
        │     points: tank_level, pump_rpm, ...
        └── press-2 (sparkplugDevice)
```

- **NBIRTH and DBIRTH** create or refresh the group, edge node, and device nodes
  and write one point per metric. A birth after a gateway restart refreshes the
  existing nodes rather than duplicating them, and tags or descriptions you have
  set on them survive.
- **NDATA and DDATA** arrive as point updates carrying the payload timestamp.
  Metrics are referenced by numeric alias after birth, and the alias assignments
  are kept on the edge node, so data that arrives after a restart resolves
  straight away. When there is no mapping for an alias -- data from a gateway
  that was already running when Simple IoT started, for instance -- Simple IoT
  requests a rebirth and the structure builds itself from the answer.
- **NDEATH and DDEATH** mark the node offline rather than deleting it. An edge
  node death takes its devices offline with it.

Each auto-created node carries a tag naming its Sparkplug identity --
`sparkplugGroup`, `sparkplugNode`, `sparkplugDevice` -- so queries select on the
structure the same way they select on a hand-set tag, and
[tag inheritance](database.md#tags) carries a site tag on the `mqtt` node down
through all of it.

Metric names become point keys, with any character a subject cannot carry
replaced by an underscore. Sparkplug types map to point types the same way other
PLC values do: see the [data types table](plc.md#data-types). Metrics carrying a
dataset, a template, or a file are skipped for now, and the rest of the message
is used. Acting as a Sparkplug primary host application (the STATE topic) and
publishing Simple IoT data outbound as Sparkplug are not part of this support.

## A multi-site deployment

Fifteen sites, each with one or more gateways publishing JSON to the broker
built into a central instance. Put identity in the topic and configure the
gateways to match:

```
{site}/{gateway}/{device}/{measurement}
```

With one `mqtt` node and a topic schema, the whole fleet needs no per-site
configuration; sites, gateways, and devices appear as they publish, each
carrying its tags:

```yaml
apiVersion: 1
nodes:
  - mqtt:
      description: Plant data
      uri: ""
      topicSchema: "{site}/{gateway}/{device}"
```

When a site needs curated metadata, give it a provisioning file instead, with a
group node per site carrying a `site` tag and explicit `mqttSub` entries below
it:

```yaml
apiVersion: 1
nodes:
  - group:
      description: Plant 07
      tag:
        site: plant-07
      children:
        - mqtt:
            description: Kepware line 3
            uri: ""
            tag:
              gateway: kepware-l3
            children:
              - mqttSub:
                  description: Tank level
                  topic: plant-07/kepware-l3/press/tank_level
                  path: $.value
                  units: cm
                  tag:
                    machine: press-3
```

The two compose: start with the schema to see what fifteen sites are publishing,
then add `mqttSub` entries, which take precedence, for the values that need
units, scaling, or careful naming.

With `tag` listed in the Database node's [Tag Point Types](database.md#tags),
every point arrives in the time series database labeled by site, gateway, and
machine:

```
points_value{key="tank_level",
             "node.tag.site"="plant-07",
             "node.tag.gateway"="kepware-l3",
             "node.tag.machine"="press-3"}
```

A Sparkplug site is one more node with `sparkplug: true` under its site group,
and its auto-created structure inherits the same `site` tag. The
[graphing section of the PLC page](plc.md#graphing-plc-data) covers how topic
hierarchies, tags, and point keys become queryable series, and the cautions that
come with them: keep unbounded values out of tags, and settle names before
collecting history you intend to keep.

## Not yet planned in detail

- **External brokers.** The `uri` setting is reserved for connecting to an
  existing broker as a client.
- **Schema-less discovery**, for browsing what an unfamiliar broker publishes
  under a prefix when no topic convention exists.
- **Per-client credentials**, so each gateway authenticates individually and can
  be restricted to its own topics.
- **MQTT 5**, which depends on the NATS server gaining support for it.
