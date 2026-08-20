# PLCs

Simple IoT can exchange data with programmable logic controllers. There is no
single PLC client; a PLC speaks one or more protocols, and you use whichever
[client](clients.md) supports the protocol you have available. This page
describes the approaches, what each one requires on the PLC side, and which ones
are implemented today.

The examples focus on Allen-Bradley ControlLogix and CompactLogix controllers
(the Logix 5000 family), because those are the most commonly asked about, but
the same approaches apply to Siemens, Beckhoff, Omron, and others.

Approaches marked **(planned)** describe work that has not been implemented yet.
They are documented here so you can plan around them and so the design is open
for discussion on the
[community forum](https://community.tmpdir.org/c/simple-iot/5).

## Choosing an approach

| Protocol                                       | Client                                  | Status    | Tag names preserved | Work required                     |
| ---------------------------------------------- | --------------------------------------- | --------- | ------------------- | --------------------------------- |
| [Modbus](#modbus)                              | [Modbus](modbus.md)                     | Available | No                  | PLC-side Modbus server            |
| [MQTT](#mqtt-planned)                          | [MQTT](mqtt.md)                         | (planned) | Yes                 | A gateway that publishes PLC tags |
| [Sparkplug B](#sparkplug-b-planned)            | [MQTT](mqtt.md)                         | (planned) | Yes                 | A gateway that speaks Sparkplug   |
| [OPC UA](#opc-ua-planned)                      | OPC UA                                  | (planned) | Yes                 | Enable the server on the PLC      |
| [EtherNet/IP](#ethernetip-tags-planned)        | Logix                                   | (planned) | Yes                 | None beyond network access        |
| [Anything else](#writing-your-own-integration) | A process of your own over the NATS API | Available | Depends             | A process you write               |

A few questions usually settle the choice:

- **Does the PLC already publish to a broker or historian?** If a gateway such
  as Kepware or Ignition is already installed and licensed, MQTT reuses it. You
  do not need to add a broker to go this route, since Simple IoT can serve MQTT
  itself.
- **Do you want Simple IoT to be the edge gateway?** If so, reading tags
  directly over OPC UA or EtherNet/IP avoids a second box and a second license.
- **How many values, and how often do they change?** A handful of stable values
  is a good fit for Modbus. Hundreds of values, or a tag list that changes as
  the PLC program is edited, is not.
- **What firmware are the Logix controllers on?** From v36 they include an OPC
  UA server, which changes the answer considerably. See
  [OPC UA](#opc-ua-planned).
- **Is the controller an open Linux platform?** Products such as Opto 22 groov
  and Phoenix Contact PLCnext publish MQTT themselves and can run Simple IoT on
  the controller. See [other controllers](#other-controllers).
- **Does the site already run Ignition?** If so, taking data across that
  boundary is often less work than connecting to each controller again. See
  [Ignition](#ignition).

## Modbus

This works today and needs no additional Simple IoT code. See the
[Modbus page](modbus.md) for how to configure a bus and its IOs.

Logix controllers do not act as Modbus servers out of the box, so the work is on
the PLC side. Common options:

- **Add-on instructions using the controller's socket object.** Rockwell
  publishes sample add-on instructions that implement Modbus TCP on the embedded
  Ethernet port of CompactLogix 5370/5380 and ControlLogix 5580 controllers. The
  instruction maps an array or user-defined type to a block of Modbus registers.
  No extra hardware, but it consumes controller scan time and socket resources.
- **A backplane communication module**, such as those from ProSoft, that
  presents a Modbus TCP or RTU interface and exchanges data with the controller
  over the backplane.
- **A standalone protocol gateway** that speaks EtherNet/IP on one side and
  Modbus on the other.

Once the PLC serves Modbus, add a `modbus` node in Simple IoT with `protocol`
set to `TCP`, `clientServer` set to `client`, and `uri` set to the controller or
gateway address. Add one `modbusIo` child for each value:

```yaml
nodes:
  - modbus:
      clientServer: client
      description: Line 3 controller
      pollPeriod: 1000
      protocol: TCP
      timeout: 500
      uri: 192.168.1.50
      children:
        - modbusIo:
            address: 100
            dataFormat: float32
            description: Tank level
            id: 1
            modbusIoType: modbusHoldingRegister
            readOnly: 1
            scale: 1
            units: cm
```

### What to plan for

Modbus carries register numbers rather than tag names, so a few constraints
follow from the protocol itself:

- **You maintain a register map by hand** in both the PLC program and the Simple
  IoT configuration. Nothing detects when the two drift apart, so treat the map
  as part of the PLC program's documentation and review it whenever the program
  changes.
- **32-bit values occupy two registers**, and the word order varies between
  implementations. If a `float32` or `int32` reads as an implausible number, try
  the swapped data format.
- **Strings, arrays, and user-defined types do not map cleanly.** Flatten what
  you need into individual registers on the PLC side.
- **Reads are polled.** Set `pollPeriod` to the slowest rate that still meets
  your needs, since every IO is read on every cycle.

Modbus suits a stable set of tens of values. Beyond that, the register map
becomes the limiting factor.

## MQTT (planned)

MQTT is the most common way to get data out of a plant network and into
something else, and it is the transport underneath Sparkplug B, described below.
Three pieces are involved:

1. Something on the PLC side that reads tags and publishes them.
2. A broker. Simple IoT provides this itself, described next.
3. A mapping from published messages into Simple IoT points.

### The broker is already built in

Simple IoT embeds a NATS server, and NATS includes an MQTT server. It needs
JetStream, which Simple IoT already runs, so exposing it is a matter of opening
a port rather than adding a dependency. A gateway or sensor then publishes
directly to Simple IoT, with no Mosquitto, HiveMQ, or EMQX to deploy, secure,
and update. On an edge device that is one fewer process to keep running.

The configuration is a port setting alongside the existing NATS port options,
disabled by default:

```
SIOT_NATS_MQTT_PORT=1883
```

Points worth knowing about the NATS MQTT server:

- It implements **MQTT 3.1.1**. Clients that require MQTT 5 are refused, which
  matters mostly for newer gateways that default to version 5.
- QoS 0, 1, and 2 are supported. Sessions and retained messages are stored in
  JetStream, so they survive a restart.
- Published messages become NATS subjects, so anything already connected to
  Simple IoT over NATS can see them. Topic levels convert as `/` to `.`, and a
  literal `.` in a topic converts to `//`. A Sparkplug topic of
  `spBv1.0/plant/DDATA/line3/tank` therefore arrives on the NATS subject
  `spBv1//0.plant.DDATA.line3.tank`.
- MQTT connections authenticate with the Simple IoT auth token, supplied in the
  password field of the connect packet. Use TLS whenever the connection leaves a
  trusted network.

An external broker still makes sense when the plant already runs one, when you
need to bridge several sites, or when you need broker features such as
clustering or fine-grained access control. Connecting to an external broker as a
client is planned as well, so the choice stays open.

### Getting the data out of the PLC

Logix controllers do not publish MQTT in a form worth depending on, so a gateway
reads tags over EtherNet/IP and republishes them. Products in common use include
Kepware's IoT Gateway, Ignition Edge with Cirrus Link MQTT Transmission,
HighByte Intelligence Hub, FactoryTalk Edge Gateway, and Opto 22 groov EPIC. All
are licensed products and become a second system to maintain, which is the main
argument for reading tags directly, described in the next section.

### Turning messages into points (planned)

A subscription node maps a topic to points. Leaving the broker address blank
would mean the server built into this instance:

```yaml
# planned, subject to change
nodes:
  - mqtt:
      description: Plant data
      uri: "" # blank uses the built-in MQTT server
      disabled: 0
      children:
        - mqttSub:
            description: Tank level
            topic: plant/line3/tank/level
            path: $.value
            units: cm
```

JSON payloads are the first target, since they cover the AWS IoT, Azure IoT, and
gateway-defined formats that most installations use.
[ADR-8](../adr/8-iot-data-models.md) compares these payload formats against the
Simple IoT point model.

### Do topics become nodes automatically? (planned)

Not by default for plain MQTT, and yes for Sparkplug B. The difference is
whether the data describes itself.

A plain MQTT topic tree looks like it should map onto the node graph, and
sometimes it does. But nothing in the protocol says which topic level is a
device and which is a measurement, or what the payload contains. Subscribing to
a wildcard on a busy plant broker and creating a node for everything that
arrives would fill the store with nodes nobody asked for, and
[synchronization](sync.md) would carry them upstream. So the default is that you
name the topics you want and how to map their payloads, as in the example above.

Discovery is still useful for finding out what a broker is publishing, so the
plan is to offer it as an option on the MQTT node rather than as the default
behavior. Turned on, it would create a child node for each topic seen under a
prefix you specify, and you keep the ones you want and delete the rest. This
follows what the [Shelly client](shelly.md) already does when it finds devices
on the network. A topic that stops publishing would be marked offline rather
than deleted, since a quiet sensor and a removed sensor look the same from
outside.

Sparkplug B is a different situation, and this is a large part of why it exists.
An edge node announces itself with a birth certificate that lists every metric
with its name and data type, and the topic namespace already separates the
group, the edge node, and the device. There is no guessing involved, so building
the node structure automatically is the intended behavior: a group becomes a
node, edge nodes and devices become nodes beneath it, and metrics become points.

## Sparkplug B (planned)

[Sparkplug B](https://sparkplug.eclipse.org/) is an Eclipse specification that
adds a defined topic namespace, a protobuf payload, and a state model on top of
MQTT. It is widely used in Industry 4.0 installations, and most of the gateways
listed above speak it, so it is the format you are most likely to meet in a
plant that has already done this work.

What it adds over plain MQTT:

- **A defined topic namespace**,
  `spBv1.0/{group}/{message type}/{edge node}/{device}`, so the structure of the
  plant is carried in the topic rather than agreed on privately between the
  publisher and each consumer.
- **Birth and death certificates.** An edge node publishes an NBIRTH listing
  every metric it will report, with names, data types, and initial values, and
  registers a death certificate with the broker so consumers learn immediately
  when it drops off. This means a consumer that connects later can discover the
  full tag list rather than guessing from traffic.
- **Report by exception with aliases.** After the birth message, values are sent
  on change and referenced by a numeric alias rather than the full name, which
  keeps the data volume low on constrained links.
- **A defined state model** for primary host applications, so publishers know
  whether the consumer that matters is online.

The structure maps onto the Simple IoT graph directly: a group becomes a node,
each edge node and device becomes a node beneath it, and each metric becomes a
point or a child node. Because a birth certificate enumerates the metrics,
Simple IoT can build that structure as edge nodes announce themselves rather
than having you configure it, which is the same idea as browsing the tag list of
a Logix controller.

Sparkplug support is planned as a phase after plain JSON payloads, because it
requires meaningfully more work: protobuf payload decoding, a cache mapping
aliases back to metric names per edge node, handling rebirth requests when that
cache is lost, and honoring the state topic if Simple IoT acts as a primary host
application. Running it against the built-in MQTT server described above removes
the broker from that list, but the payload and state handling remain.

## OPC UA (planned)

[OPC UA](https://opcfoundation.org/about/opc-technologies/opc-ua/) (IEC 62541)
is the vendor-neutral standard for industrial data exchange, and it is the
single client that would cover the widest range of hardware. It is not
implemented yet.

### What already has an OPC UA server

Most modern controllers, and this now includes Allen-Bradley:

- **Logix 5380, 5580, and 5590 controllers have a native OPC UA server from
  firmware v36**, disabled by default and enabled in the controller
  configuration. If your controllers are on v36 or later, this is the most
  direct path available today for reading tags by name, with no add-on
  instruction, gateway, or license involved.
- **CompactLogix 5480** hosts FactoryTalk Linx Gateway on its Windows side to
  serve OPC UA. Older Logix controllers need FactoryTalk Linx Gateway or a
  product such as Kepware.
- **Siemens** S7-1200 and S7-1500 include a server, as do **Phoenix Contact
  PLCnext**, **Beckhoff TwinCAT**, and **B&R** controllers.
- **Ignition** and most gateway products expose one as well, so OPC UA is often
  a way to reach data that has already been collected.

### Why it fits Simple IoT well

- **One client covers many vendors.** Every other option on this page is
  specific to a protocol or a product line.
- **The address space is browsable**, and carries tag names, data types, and
  engineering units. Simple IoT can create the node structure by browsing the
  server, the same idea as a Sparkplug birth certificate or a Logix tag list,
  rather than having you type node IDs.
- **Subscriptions rather than polling.** You register the values you care about
  with a publishing interval and an optional deadband, and the server sends
  changes. This scales considerably better than a poll loop.
- **Values arrive with context.** Each update carries a source timestamp and a
  status code, so a stale or bad reading is distinguishable from a good one.
  Points already carry a timestamp; representing status is a question to settle
  when the client is built.

### What it would look like

```yaml
# planned, subject to change
nodes:
  - opcua:
      description: Line 3 controller
      endpoint: opc.tcp://192.168.1.50:4840
      securityPolicy: Basic256Sha256
      securityMode: SignAndEncrypt
      publishInterval: 1000
      disabled: 0
      children:
        - opcuaNode:
            description: Tank level
            nodeId: ns=2;s=Tank_Level_PV
            deadband: 0.5
            scale: 1
            offset: 0
            units: cm
```

[gopcua](https://github.com/gopcua/opcua) is the likely dependency. It is a
native Go implementation with browsing, subscriptions, and the standard security
policies, it is used in production elsewhere including Telegraf, and staying in
pure Go keeps the single static binary and the ARM builds intact.

### The part that takes the work

Security is where OPC UA costs more than the other options. A client presents an
application instance certificate, and the server has to be told to trust it,
which is usually a manual step in the server's own configuration. On top of that
sit the security policy, the message mode, and the user token. Simple IoT would
need to generate and store a certificate, present it, and give you a way to see
why a connection was rejected, since a rejected certificate and a wrong password
look similar from the client side.

Anonymous connections with no security are common on isolated plant networks and
are the quickest way to get a first reading, but they are not a good place to
stop. See the [security reference](../ref/security.md) for how Simple IoT
handles certificates elsewhere.

### Beyond reading values

A useful client is browse, read, write, and subscribe. The rest of the
specification is much larger, and none of it is needed to get data flowing:

- **Historical access**, for pulling archived values out of a server that keeps
  them.
- **Methods**, for calling functions the server exposes.
- **Alarms and events**, which would map onto [notifications](notifications.md).
- **OPC UA PubSub**, which publishes over MQTT or UDP rather than a client
  session. Since Simple IoT already contains an MQTT server, receiving PubSub
  data would build on the same work as the MQTT client.

Serving OPC UA is the other direction worth considering. Exposing Simple IoT
nodes as an address space would let existing SCADA software read Simple IoT data
without any of it needing to know what Simple IoT is.

## EtherNet/IP tags (planned)

Reading Logix tags over EtherNet/IP means no gateway, no license, and no
register map. It is not implemented yet.

This overlaps with [OPC UA](#opc-ua-planned), which reaches the same tags on
firmware v36 and later through a standard that also covers other vendors. Where
an EtherNet/IP client still earns its place is on controllers older than v36,
which is a large installed base, and on sites that would rather not enable
another server on the controller.

The intended design is a `logix` node holding the controller connection, with a
child node per tag:

```yaml
# planned, subject to change
nodes:
  - logix:
      description: Line 3 controller
      uri: 192.168.1.50
      path: "1,0" # backplane slot
      pollPeriod: 1000
      disabled: 0
      children:
        - logixTag:
            description: Tank level
            tag: Tank_Level_PV
            scale: 1
            offset: 0
            units: cm
```

Because Logix controllers can report their own tag list, Simple IoT could browse
the controller and create the child nodes for you, rather than having you type
each tag name. That is the part that makes this approach meaningfully better
than the alternatives.

Implementation notes for anyone interested in helping:

- [gologix](https://github.com/danomagnum/gologix) is a pure Go implementation
  of the Logix CIP services and is the likely dependency. Staying in pure Go
  keeps the single statically linked binary and the cross-compilation story
  intact.
- [libplctag](https://github.com/libplctag/libplctag) is more widely proven but
  is a C library, so using it would require cgo and give up the above.
- Reading many tags in one multi-service request matters for performance;
  reading them one at a time does not scale past a few dozen.
- Controllers limit the number of concurrent connections, so one connection per
  `logix` node is the right granularity.

## Writing to a PLC

Simple IoT clients use a `valueSet` point to request a change and a `value`
point to report what was read back, so writes fit the existing pattern in every
approach above. The Modbus client supports this today through the `readOnly`
setting on each IO.

Writing into a running machine deserves a conversation with whoever owns it.
Consider leaving IOs read-only unless a write is genuinely required, and putting
range and interlock checks in the PLC program rather than relying on the value
sent from outside.

## Data types

Each point carries a data type along with its data, so a PLC value keeps the
shape it had in the controller rather than being flattened into a single numeric
field. The types currently defined are float, int, string, and JSON, and the set
can be extended when a PLC type needs a representation that does not fit the
existing ones.

| PLC type          | Point data type                                       |
| ----------------- | ----------------------------------------------------- |
| BOOL              | int, 0 or 1                                           |
| SINT, INT, DINT   | int                                                   |
| REAL              | float                                                 |
| STRING            | string                                                |
| Arrays            | One point per element, distinguished by the point key |
| User-defined type | One point per member, or a child node (planned)       |

The `scale` and `offset` fields convert raw values into engineering units:
`value = raw * scale + offset`. They apply to numeric types.

Deciding how to represent user-defined types is the open question. A flat
structure, with one point per member named by the member path, keeps the CRDT
properties that synchronization depends on and is the likely starting point. A
JSON point is available for cases where the structure is better kept intact, at
the cost of merging the whole value as a unit.
[ADR-1](../adr/1-consider-changing-point-data-type.md) covers the reasoning
behind the point data types.

### Text data

Most of what a PLC reports is numeric, but text appears often enough to plan
for. Where it shows up is fairly consistent across plants: batch, lot, recipe,
and part numbers; barcode and RFID reads; serial numbers being recorded for
traceability; operator or badge IDs; work order numbers; machine state and alarm
text; and firmware or program version strings.

Logix controllers have a `STRING` type, which is a structure holding a length
and a character array, and OPC UA and Sparkplug B both carry strings natively.
Modbus does not have a string type at all, so devices that report one pack ASCII
into consecutive registers, two characters per register, by local convention.

The useful observation is that this text is almost always **identity or context
for the numeric data rather than a measurement in its own right**. Nobody graphs
a batch number; they want to know which batch a temperature trace belongs to.
That distinction decides where it should go:

- **Text that changes slowly and describes the thing producing data** belongs on
  a tag point, where it becomes a label on every point emitted beneath it. A
  line, a machine, or an installed product variant fits here.
- **Text that changes with production** is where care is needed. A batch number
  as a tag gives you exactly the query you want, at the cost of a new series per
  batch. That is affordable for batches lasting hours and not affordable for
  something changing every few seconds.
- **State and status** are better stored as numbers with a lookup. A machine
  state written as an integer graphs and alarms cleanly, and Grafana value
  mappings display the names. Historians have handled enumerations this way for
  a long time, and it is worth doing even when the PLC has the state as a
  string.
- **Alarm and event text** is a poor fit for a time series database in any form.
  In Simple IoT it maps better onto [notifications](notifications.md).

Text is stored in points and kept in the store regardless, so the current value
is visible in the UI and available over the [API](../ref/api.md), and its
history is in the [store](../ref/store.md). What text does not do today is reach
VictoriaMetrics, which converts non-numeric values to zero, so the Database
client skips string points rather than filling the database with zeros.

If a string has to be queryable in VictoriaMetrics and does not suit a tag, the
established pattern is an information series: a value of 1 carrying the string
as a label, joined onto the real measurement at query time with `group_left`.
That is worth knowing about, though for most PLC data a tag point is the simpler
answer.

## Graphing PLC data

None of the clients on this page write to a time series database themselves.
They create nodes and publish points, and a [Database node](database.md) writes
those points to VictoriaMetrics or InfluxDB, where [Grafana](graphing.md) reads
them. So the question of how a PLC tag or an MQTT topic ends up as a queryable
series is really a question about what nodes and points the client creates, and
the answer is the same whichever protocol brought the data in.

### What a point becomes

Every point written to VictoriaMetrics arrives as the metric `points_value`,
with these labels:

| Label              | Comes from                                     |
| ------------------ | ---------------------------------------------- |
| `type`             | The point type, such as `value`                |
| `key`              | The point key, used for arrays and maps        |
| `node.id`          | The node that emitted the point                |
| `node.type`        | The node type, such as `modbusIo` or `mqttSub` |
| `node.description` | The node's Description field                   |
| `node.tag.*`       | Tag points on the node, and on its ancestors   |

That last row is what makes plant structure queryable. A tag point set once on a
node is inherited by every point emitted beneath it, up to the Database node's
parent, so a site or line or machine label is set in one place rather than
repeated on every sensor. The [database page](database.md#tags) covers the
rules, including that the value nearest the emitting node wins.

### How this is usually done elsewhere

The common tool for MQTT into a time series database is Telegraf's
`mqtt_consumer` input, and it does two things worth knowing:

- **It stores the whole topic as a tag named `topic`, by default.** You can turn
  that off with `topic_tag = ""`.
- **It also parses the topic into separate tags**, through `topic_parsing` rules
  that assign each topic level to a measurement, a tag, or a field.

InfluxData's own guidance is that the whole topic on its own needs extra
processing before it is useful, and that parsing the levels into distinct tags
is what makes the data queryable. Other tools take the same approach by
different means; `mqtt2prometheus`, for example, pulls labels out of the topic
with a regular expression.

So the answer to whether you store the entire topic is usually "yes, and also
the parsed levels". The full topic is worth keeping for tracing a series back to
its source and for selecting one exact series. It is not a good primary label,
because a query that wants every tank level on line 3 cannot express that
against an opaque string.

### How it maps in Simple IoT

The topic hierarchy becomes the node hierarchy, and the levels you want to query
on become tag points. Given a topic of `plant-a/line3/press/tank_level` carrying
`{"value": 42.1}`:

```
plant-a              tag: site=plant-a
└── line3            tag: line=3
    └── press        tag: machine=press-3
        └── Tank level   (mqttSub, topic: plant-a/line3/press/tank_level)
                         tag: topic=plant-a/line3/press/tank_level
```

The point emitted by that subscription node is written as:

```
points_value{type="value",
             "node.description"="Tank level",
             "node.tag.site"="plant-a",
             "node.tag.line"="3",
             "node.tag.machine"="press-3",
             "node.tag.topic"="plant-a/line3/press/tank_level"}
```

Label names contain periods, so MetricsQL queries quote them. Every tank level
on line 3, regardless of machine:

```
points_value{type="value", "node.tag.line"="3", "node.description"="Tank level"}
```

Storing the full topic as a tag point named `topic` is the same idea as
Telegraf's default, and it needs nothing new in the Database client, since it is
an ordinary tag point.

### Choosing the point shape

Two arrangements both work, and the payload usually decides:

- **A node per measurement**, with a point of type `value`. This matches how
  [Modbus IOs](modbus.md) already work, and suits topics that carry a single
  scalar.
- **A node per device**, with one point per field, where the point key holds the
  field name. A payload with twenty fields becomes one node and twenty points
  rather than twenty nodes, and the field name is queryable as the `key` label:

  ```
  points_value{type="value", key="tank_level", "node.tag.machine"="press-3"}
  ```

For [Sparkplug B](#sparkplug-b-planned) the structure is already decided by the
specification: the group, edge node, and device become nodes, which means they
become inherited tags without any configuration, and each metric becomes a
point.

### Things to plan for

- **Keep unbounded values out of tags.** Each distinct combination of labels is
  a separate series, and series count is what makes a time series database slow,
  not label length. A topic level holding a message ID or a timestamp should not
  become a tag, and where topics carry one, storing the full topic is a poor
  idea as well.
- **Settle tag names before collecting history.** Editing a tag or a description
  starts a new series from that moment, and a query spanning the change sees
  both.
- **Publish numbers, not strings.** VictoriaMetrics converts non-numeric values
  to zero, so the Database client skips string points entirely.

## Other controllers

The Logix approach above assumes a closed controller that you can only reach
over the network. Several popular platforms are more open than that, which
changes what is worth doing.

Any controller that serves Modbus TCP or RTU works today with no additional
code, which includes most Siemens, Omron, Schneider, and WAGO products, either
natively or through a communication module.

### Opto 22 groov EPIC and groov RIO

These are among the easiest controllers to work with, because the protocols are
in the firmware rather than in a separate gateway:

- A **Modbus/TCP server** is available out of the box, so the Modbus client
  works with them today.
- **MQTT with Sparkplug B or string payloads** is built into the firmware and
  configured from groov Manage, with no gateway software or license involved.
  Combined with the MQTT server built into Simple IoT, a groov RIO can publish
  straight into Simple IoT with nothing in between.
- A **REST API** covers the I/O channels, with a Swagger document built into the
  device. A process of your own can poll it and publish points today, as
  described below.
- The controller **runs Linux**, and a free shell license enables SSH, with
  container support on firmware 4.0.0 and later. See
  [running Simple IoT on the controller](#running-simple-iot-on-the-controller).

Opto 22 treats shell access as an advanced, self-supported option, so weigh that
against how much you value keeping everything on one device.

### Phoenix Contact PLCnext

PLCnext controllers run Linux alongside the IEC 61131 runtime and are designed
for adding your own software:

- **Modbus TCP and OPC UA** are available, so Modbus works with Simple IoT
  today.
- A **gRPC data interface** exposes the Global Data Space, so an external
  program can read and write controller variables by name. Phoenix Contact
  publishes the protocol definitions, and Go is a first-class gRPC language, so
  a PLCnext client would be similar in shape to the Logix client described above
  and would read tags by name rather than by register. This is a reasonable
  candidate once the Logix client exists (planned).
- Container support has been available since firmware 2020.0, and the
  controllers are ARM-based, so Simple IoT can run on the device.

### Running Simple IoT on the controller

Both platforms above run Linux on ARM and allow you to install your own
software, which opens an option that a Logix controller does not. Simple IoT is
a single statically linked binary with no runtime dependencies, and
`siot_build_arm` and `siot_build_arm64` produce builds for these processors, so
it can run on the controller itself rather than on a separate computer beside
it.

That removes the network hop: read process data through the local interface,
store points on the device, and [synchronize](sync.md) upstream when a
connection is available. A few things to check before committing to it:

- The vendor's support policy for running your own software.
- Available flash and its endurance, since the store writes to disk. The
  [store reference](../ref/store.md) covers the settings that affect this.
- What a firmware update does to anything you installed.

### Siemens

The S7 protocol is not implemented. Siemens S7-1200 and S7-1500 controllers
include an OPC UA server, so the [OPC UA client](#opc-ua-planned) would cover
them without anything S7-specific. Until then, Modbus or the custom client
approach below covers these cases.

## Ignition

[Ignition](https://inductiveautomation.com/) is a SCADA platform rather than a
PLC, but it comes up often enough to be worth its own section: many plants
already run it, and it is frequently the system that already has a connection to
every controller on the floor. Where that is true, integrating with Ignition is
usually less work than connecting to each PLC again.

Sparkplug B is the natural boundary between the two systems, and the Cirrus Link
modules move data in both directions:

- **MQTT Transmission** publishes Ignition tags, including tags it reads from
  Logix and other controllers over OPC UA, as Sparkplug B. Simple IoT would
  subscribe to those (planned), and with the MQTT server built into Simple IoT
  it can be the broker that Transmission publishes to, so no separate broker is
  required.
- **MQTT Engine** subscribes to a broker and turns Sparkplug messages into
  Ignition tags. If Simple IoT publishes Sparkplug (planned), data from Simple
  IoT nodes appears in Ignition alongside everything else, which is a practical
  way to get edge data onto existing screens and into existing alarm
  configurations.

Ignition also exposes an OPC UA server, so an OPC UA client would be another
path to the same data. Ignition Edge runs on hardware such as groov EPIC, which
is worth knowing if you are choosing between running Ignition Edge and Simple
IoT on the same device, or running both.

The two systems solve different problems and coexist well. Ignition is typically
the plant-floor HMI and SCADA layer, while Simple IoT handles distributed state
and configuration and synchronizes it between the edge and the cloud. Sending
data across the boundary as Sparkplug lets each do what it is good at.

## Writing your own integration

If none of the above fits, you can connect a process of your own to the Simple
IoT [NATS API](../ref/api.md) and publish points into the store. That process
can be written in any language with a NATS client, and can use whatever PLC
library suits it. This is often the fastest path for a one-off protocol, and it
keeps the protocol-specific code out of your Simple IoT deployment. See the
[integration page](../ref/integration.md) for the available integration points.

If the result is generally useful, consider contributing it as a client instead.
The [client reference](../ref/client.md) describes what that involves.
