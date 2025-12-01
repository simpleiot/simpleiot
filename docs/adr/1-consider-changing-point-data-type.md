# Point Data Type Changes

- Author: Cliff Brake Last updated: 2023-06-13
- Issue at: https://github.com/simpleiot/simpleiot/issues/254
- PR/Discussion:
  - https://github.com/simpleiot/simpleiot/pull/279
  - https://github.com/simpleiot/simpleiot/pull/565
  - https://github.com/simpleiot/simpleiot/pull/566
- Status: Review

**Contents**

<!-- toc -->

## Problem

The current point data type is fairly simple and has proven useful and flexible
to date, but we may benefit from additional or changed fields to support more
scenarios. It seems in any data store, we need at the node level to be able to
easily represent:

1. arrays
1. maps

IoT systems are distributed systems that evolve over time. If can't easily
handle schema changes and synchronize data between systems, we don't have
anything.

## Context/Discussion

Should we consider making the `point` struct more flexible?

The reason for this is that it is sometimes hard to describe a
sensor/configuration value with just a few fields.

### Requirements

- IoT systems are often connected by unreliable networks (cellular, etc). All
  devices/instances in a SIOT should be able to functional autonomously (run
  rules, etc) and then synchronize again when connected.
- all systems must converge to the same configuration state. We can probably
  tolerate some lost time series data, but configuration and current state must
  converge. When someone is remotely looking at a device state, we want to make
  sure they are seeing the same things a local operator is seeing.

### evolvability

From Martin Kleppmann's book:

> In a database, the process that writes to the database encodes the data, and
> the process that reads from the database decodes it. There may just be a
> single process accessing the database, in which case the reader is simply a
> later version of the same process—in that case you can think of storing
> something in the database as sending a message to your future self.
>
> Backward compatibility is clearly necessary here; otherwise your future self
> won’t be able to decode what you previously wrote.
>
> In general, it’s common for several different processes to be accessing a
> database at the same time. Those processes might be several different
> applications or services, or they may simply be several instances of the same
> service (running in parallel for scalability or fault tolerance). Either way,
> in an environment where the application is changing, it is likely that some
> processes accessing the database will be running newer code and some will be
> running older code—for example because a new version is currently being
> deployed in a rolling upgrade, so some instances have been updated while
> others haven’t yet.
>
> This means that a value in the database may be written by a newer version of
> the code, and subsequently read by an older version of the code that is still
> running. Thus, forward compatibility is also often required for databases.
>
> However, there is an additional snag. Say you add a field to a record schema,
> and the newer code writes a value for that new field to the database.
> Subsequently, an older version of the code (which doesn’t yet know about the
> new field) reads the record, updates it, and writes it back. In this
> situation, the desirable behavior is usually for the old code to keep the new
> field intact, even though it couldn’t be interpreted.
>
> The encoding formats discussed previously support such preservation of unknown
> fields, but sometimes you need to take care at an application level, as
> illustrated in Figure 4-7. For example, if you decode a database value into
> model objects in the application, and later re-encode those model objects, the
> unknown field might be lost in that translation process. Solving this is not a
> hard problem; you just need to be aware of it.

Some discussion of this book:
https://community.tmpdir.org/t/book-review-designing-data-intensive-applications/288/6

### CRDTs

Some good talks/discussions:

> I also agree CRDTs are the future, but not for any reason as specific as the
> ones in the article. Distributed state is so fundamentally complex that I
> think we actually need CRDTs (or something like them) to reason about it
> effectively. And certainly to build reliable systems. The abstraction of a
> single, global, logical truth is so nice and tidy and appealing, but it
> becomes so leaky that I think all successful systems for distributed state
> will abandon it beyond a certain scale. --
> [Peter Bourgon](https://lobste.rs/s/9fufgr/i_was_wrong_crdts_are_future)

[CRDTs, the hard parts by Martin Kleppmann](https://youtu.be/x7drE24geUw)

[Infinite Parallel Universes: State at the Edge](https://www.infoq.com/presentations/architecture-global-scale/)

[Wikipedia article](https://en.wikipedia.org/wiki/Conflict-free_replicated_data_type)

Properties of CRDTs:

- **Associative** (order in which operations are performed does matter)
- **Commutative** (changing order of operands does not change result)
- **Idempotent** (operation can be applied multiple times without changing the
  result, tolerate over-merging)

The existing SIOT Node/Point data structures were created before I know what a
CRDT was, but they happen to already give a node many of the properties of a
CRDT -- IE, they can be modified independently, and then later merged with a
reasonable level of conflict resolution.

For reliable data synchronization in distributed systems, there has to be some
metadata around data that facilitates synchronization. This can be done in two
ways:

1. add meta data in parallel to the data (turn JSON into a CRDT, example
   [automerge](https://github.com/automerge/automerge) or
   [yjs](https://docs.yjs.dev/))
2. express all data using simple primitives that facilitate synchronization

Either way, you have to accept constraints in your data storage and transmission
formats.

To date, we have chosen to follow the 2nd path (simple data primitives).

### Operational transforms

There are two fundamental schools of thought regarding data synchronization:

1. Operation transforms. In this method, a central server arbitrates all
   conflicts and hands the result back to other instances. This is an older
   technique and is used in applications like Google docs.
2. CRDTs -- this is a newer technique that works with multiple network
   connections and does not require a central server. Each instance is capable
   of resolving conflicts themselves and converging to the same point.

While a classical OT arrangement could probably work in a traditional SIOT
system (where all devices talk to one cloud server), it would be nice if we are
not constrained to this architecture. This would allow us to support peer
synchronization in the future.

### Other Standards

Some reference/discussion on other standards:

#### Sparkplug

https://github.com/eclipse/tahu/blob/master/sparkplug_b/sparkplug_b.proto

The sparkplug data type is huge and could be used to describe very complex data.
This standard came out of the industry 4.0 movement where a factory revolves
around a common MQTT messaging server. The assumption is that everything is
always connected to the MQTT server. However, with complex types, there is no
provision for intelligent synchronization if one system is disconnected for some
amount of time -- its all or nothing, thus it does not seem like a good fit for
SIOT.

#### SenML

https://datatracker.ietf.org/doc/html/draft-ietf-core-senml-08#page-9

#### tstorage

The tstorage Go package has
[an interesting data storage type](https://community.tmpdir.org/t/the-tstorage-time-series-package-for-go/331):

```go
type Row struct {
	// The unique name of metric.
	// This field must be set.
	Metric string
	// An optional key-value properties to further detailed identification.
	Labels []Label
	// This field must be set.
	DataPoint
}

type DataPoint struct {
	// The actual value. This field must be set.
	Value float64
	// Unix timestamp.
	Timestamp int64
}

type Label struct {
	Name  string
	Value string
```

In this case there is one value and an array of labels, which are essentially
key/value strings.

#### InfluxDB

InfluxDB's line protocol contains the following:

```go
type Metric interface {
	Time() time.Time
	Name() string
	TagList() []*Tag
	FieldList() []*Field
}

type Tag struct {
	Key   string
	Value string
}

type Field struct {
	Key   string
	Value interface{}
}
```

where the Field.Value must contain one of the InfluxDB supported types (bool,
uint, int, float, time, duration, string, or bytes).

### time-series storage considerations

Is it necessary to have all values in one point, so they can be grouped as one
entry in a time series data base like influxdb? Influx has a concept of tags and
fields, and you can have as many as you want for each sample. Tags must be
strings and are indexed and should be low cardinality. Fields can be any
datatype influxdb supports. This is a very simple, efficient, and flexible data
structure.

### Example: location data

One system we are working with has extensive location information
(City/State/Facility/Floor/Room/Isle) with each point. This is all stored in
influx so we can easily query information for any location in the past. With
SIOT, we could not currently store this information with each value point, but
would rather store location information with the node as separate points. One
concern is if the device would change location. However, if location is stored
in points, then we will have a history of all location changes of the device. To
query values for a location, we could run a two pass algorithm:

1. query history and find time windows when devices are in a particular
   location.
1. query these time ranges and devices for values

This has the advantage that we don't need to store location data with every
point, but we still have a clear history of what data come from where.

### Example: file system metrics

When adding metrics, we end up with data like the following for disks
partitions:

```
Filesystem     Size Used Avail Use% Mounted on
tmpfs          16806068224 0 16806068224   0% /dev
tmpfs          16813735936 1519616 16812216320   0% /run
ext2/ext3      2953064402944 1948218814464 854814945280  70% /
tmpfs          16813735936 175980544 16637755392   1% /dev/shm
tmpfs          16813740032 3108966400 13704773632  18% /tmp
ext2/ext3      368837799936 156350181376 193680359424  45% /old3
msdos          313942016 60329984 253612032  19% /boot
ext2/ext3      3561716731904 2638277668864 742441906176  78% /scratch
tmpfs          3362746368 118784 3362627584   0% /run/user/1000
ext2/ext3      1968874332160 418203766784 1450633895936  22% /run/media/cbrake/59b35dd4-954b-4568-9fa8-9e7df9c450fc
fuseblk        3561716731904 2638277668864 742441906176  78% /media/fileserver
ext2/ext3      984372027392 339508314112 594836836352  36% /run/media/cbrake/backup2
```

It would be handy if we could store filesystem as a tag, size/used/avail/% as
fields, and mount point as text field.

We already have an array of points in a node -- can we just make one array work?
The size/used/avail/% could easily be stored as different points. The text field
would store the mount point, which would tie all the stats for one partition
together. Then the question is how to represent the filesystem? With the added
`Key` field in proposal #2, we can now store the mount point as the key.

| Type           | Key   | Text  | Value   |
| -------------- | ----- | ----- | ------- |
| filesystemSize | /home |       | 1243234 |
| filesystemUsed | /home |       | 234222  |
| filesystemType | /home | ext4  |         |
| filesystemSize | /home |       | 1000000 |
| filesystemUsed | /date |       | 10000   |
| filesystemType | /home | btrfs |         |

### Representing arrays

With the `key` field, we can represent arrays as a group of points, where key
defines the position in the array. For node points to be automatically decoded
into an array struct fields by the SIOT client manager, the key must be an
integer represented in string form.

One example where we do this is for selecting days of the week in schedule rule
conditions. The key field is used to select the weekday. So we can have a series
of points to represent Weekdays. In the below, Sunday is the 1st point set to 0,
and Monday is the 2nd point, set to 1.

```go
[]data.Point{
  {
    Type: "weekday",
    Key: "0",
    Value: 0,
  },
  {
    Type: "weekday",
    key: "1",
    Value: 0,
  },
}
```

In this case, the condition node has a series of weekday points with keys 0-6 to
represent the days of the week.

The SIOT
[data.Decode](https://pkg.go.dev/github.com/simpleiot/simpleiot/data#Decode) is
used by the client manager to initialize array fields in a client struct. The
following assumptions are made:

- the value in the `key` field is converted to an int and used as the index into
  the field array.
- if there are missing array entries, they are filled with zero values.
- the
  [data.MergePoints](https://pkg.go.dev/github.com/simpleiot/simpleiot/data#MergePoints)
  uses the same algorithm.
- if a point is inserted into the array or moved, all array points affected must
  be sent. For example, if you have an array of length 20, and you insert a new
  value at the beginning, then all 21 points must to be sent. This can have
  implications for rules or any other logic that use the Point `key` field.

This does not have perfect CRDT properties, but typically these arrays are
generally small and are only modified in one place.

If you need more advanced functionality, you can bypass the data Decode/Merge
functions and process the points manually and then use any algorithm you want to
process them.

### Point deletions

To date, we've had no need to delete points, but it may be useful in the future.

Consider the following sequence of point changes:

1. t1: we have a point
1. t2: A deletes the point
1. t3: B concurrently change the point value

The below table shows the point values over time with the current point merge
algorithm:

| Time | Value | Tombstone |
| ---- | ----- | --------- |
| t1   | 10    | 0         |
| t2   | 10    | 1         |
| t3   | 20    | 0         |

In this case, the point becomes undeleted because the last write wins (LWW). Is
this a problem? What is the desired behavior? A likely scenario is that a device
will be continually sending value updates and a user will make a configuration
change in the portal that deletes a point. Thus it seems delete changes should
always have precedence. However, with the last write wins (LWW) merge algorithm,
the tombstone value could get lost. It may make sense to:

- make the tombstone value an int
- only increment it
- when merging points, the highest tombstone value wins
- odd value of tombstone value means point is deleted

Thus the tombstone value is merged independently of the timestamp and thus is
always preserved, even if there concurrent modifications.

The following table shows the values with the modified point merge algorithm.

| Time | Value | Tombstone |
| ---- | ----- | --------- |
| t1   | 10    | 0         |
| t2   | 10    | 1         |
| t3   | 20    | 1         |

### Duration, Min, Max

The current Point data type has Duration, Min, and Max fields. This is used for
when a sensor value is averaged over some period of time, and then reported. The
Duration, Min, Max fields are useful for describing what time period the point
was obtained, and what the min/max values during this period were.

### Representing maps

In the file system metrics example below, we would like to store a file system
type for a particular mount type. We have 3 pieces of information:

```go
data.Point {
  Type: "fileSystem",
  Text: "/media/data/",
  ????: "ext4",
}
```

Perhaps we could add a key field:

```go
data.Point {
  Type: "fileSystem",
  Key: "/media/data/",
  Text: "ext4",
}
```

The `Key` field could also be useful for storing the mount point for other
size/used, etc points.

### making use of common algorithms and visualization tools

A simple point type makes it very nice to write common algorithms that take in
points, and can always assume the value is in the value field. If we store
multiple values in a point, then the algorithm needs to know which point to use.

If an algorithm needs multiple values, it seems we could feed in multiple point
types and discriminated by point type. For example, if an algorithm used to
calculate % of a partition used could take in total size and used, store each,
and the divide them to output %. The data does not necessarily need to live in
the same point. Could this be used to get rid of the min/max fields in the
point? Could these simply be separate points?

- Having min/max/duration as separate points in influxdb should not be a problem
  for graphing -- you would simply qualify the point on a different type vs
  selecting a different field.
- if there is a process that is doing advanced calculations (say taking the
  numerical integral of flow rate to get total flow), then this process could
  simply accumulate points and when it has all the points for a timestamp, then
  do the calculation.

### Schema changes and distributed synchronization

A primary consideration of Simple IoT is easy and efficient data synchronization
and easy schema changes.

One argument against embedded maps in a point is that adding these maps would
likely increase the possibility of schema version conflicts between versions of
software because points are overwritten. Adding maps now introduces a schema
into the point that is not synchronized at the key level. There will also be a
temptation to put more information into point maps instead of creating more
points.

With the current point scheme, it is very easy to synchronize data, even if
there are schema changes. All points are synchronized, so one version can write
one set of points, and another version another, and all points will be sync'd to
all instances.

There is also a concern that if two different versions of the software use
different combinations of field/value keys, there could be information lost. The
simplicity and ease of merging Points into nodes is no longer simple. As an
example:

```go
Point {
  Type: "motorPIDConfig",
  Values: {
    {"P": 23},
    {"I": 0.8},
    {"D": 200},
  },
}
```

If an instance with an older version writes a point that only has the "P" and
"I" values, then the "D" value would get lost. We could merge all maps on writes
to prevent losing information. However if we have a case where we have 3
systems:

Aold -> Bnew -> Cnew

If Aold writes an update to the above point, but only has P,I values, then this
point is automatically forwarded to Bnew, and then Bnew forwards it to Cnew.
However, Bnew may have had a copy with P,I,D values, but the D is lost when the
point is forwarded from Aold -> Cnew. We could argue that Bnew has previously
synchronized this point to Cnew, but what if Cnew was offline and Aold sent the
point immediately after Cnew came online before Bnew synchronized its point.

The bottom line is there are edge cases where we don't know if the point map
data is fully synchronized as the map data is not hashed. If we implement arrays
and maps as collections of points, then we can be more sure everything is
synchronized correctly because each point is a struct with fixed fields.

### Is there any scenario where we need multiple tags/labels on a point?

If we don't add maps to points, the assumption is any metadata can be added as
additional points to the containing node. Will this cover all cases?

### Is there any scenario where we need multiple values in a point vs multiple points?

If we have points that need to be grouped together, they could all be sent with
the same timestamp. Whatever process is using the points could extract them from
a timeseries store and then re-associate them based on common timestamps.

Could duration/min/max be sent as separate points with the same timestamp
instead of extra fields in the point?

The NATS APIs allow you to send multiple points with a message, so if there is
ever a need to describe data with multiple values (say min/max/etc), these can
simply be sent as multiple points in one message.

### Is there any advantage to flat data structures?

Flat data structures where the fields consist only of simple types (no nested
objects, arrays, maps, etc). This is essentially what tables in a relational
database are. One advantage to keeping the point type flat is it would map
better into a relational database. If we add arrays to the Point type, then it
will not longer map into a single relational database table.

## Design

### Original Point Type

```go
type Point struct {
	// ID of the sensor that provided the point
	ID string `json:"id,omitempty"`

	// Type of point (voltage, current, key, etc)
	Type string `json:"type,omitempty"`

	// Index is used to specify a position in an array such as
	// which pump, temp sensor, etc.
	Index int `json:"index,omitempty"`

	// Time the point was taken
	Time time.Time `json:"time,omitempty"`

	// Duration over which the point was taken. This is useful
	// for averaged values to know what time period the value applies
	// to.
	Duration time.Duration `json:"duration,omitempty"`

	// Average OR
	// Instantaneous analog or digital value of the point.
	// 0 and 1 are used to represent digital values
	Value float64 `json:"value,omitempty"`

	// Optional text value of the point for data that is best represented
	// as a string rather than a number.
	Text string `json:"text,omitempty"`

	// statistical values that may be calculated over the duration of the point
	Min float64 `json:"min,omitempty"`
	Max float64 `json:"max,omitempty"`
}
```

### Proposal #1

This proposal would move all the data into maps.

```go
type Point struct {
    ID string
    Time time.Time
    Type string
    Tags map[string]string
    Values map[string]float64
    TextValues map[string]string
}
```

The existing min/max would just become fields. This would map better into
influxdb. There would be some redundancy between Type and Field keys.

### Proposal #2

```go
type Point struct {
	// The 1st three fields uniquely identify a point when receiving updates
	Type string
	Key string

	// The following fields are the values for a point
	Time time.Time
	(removed) Index float64
	Value float64
	Text string
	Data []byte

	// Metadata
	Tombstone int
}
```

_Updated 2023-06-13: removed the `Index` field. We will use the `Key` field for
array indices._

Notable changes from the first implementation:

- removal of the `ID` field, as any ID information should be contained in the
  parent node. The `ID` field is a legacy from 1-wire setups where we
  represented each 1-wire sensor as a point. However, it seems now each 1-wire
  sensor should have its own node.
- addition of the `Key` field. This allows us to represent maps in a node, as
  well as add extra identifying information for a point.
- the `Point` is now identified in the merge algorithm using the `Type` and
  `Key`. Before, the `ID`, `Type`, and `Index` were used.
- the `Data` field is added to give us the flexibility to store/transmit data
  that does not fit in a Value or Text field. This should be used sparingly, but
  gives us some flexibility in the future for special cases. This came out of
  some comments in an Industry 4.0 community -- basically types/schemas are good
  in a communication standard, as long as you also have the capability to send a
  blob of data to handle all the special cases. This seems like good advice.
- the `Tombstone` fields is added as an `int` and is always incremented. Odd
  values of `Tombstone` mean the point was deleted. When merging points, the
  highest tombstone value always wins.

## Decision

Going with proposal #2 -- we can always revisit this later if needed. This has
minimal impact on the existing code base.

## Objections/concerns

(Some of these are general to the node/point concept in general)

- Q: _with the point datatype, we lose types_
  - A: in a single application, this concern would perhaps be a high priority,
    but in a distributed system, data synchronization and schema migrations must
    be given priority. Typically these collections of points are translated to a
    type by the application code using the data, so any concerns can be handled
    there. At least we won't get JS undefined crashes as Go will fill in zero
    values.
- Q: _this will be inefficient converting points to types_
  - A: this does take processing time, but this time is short compared to the
    network transfer times from distributed instances. Additionally,
    applications can cache nodes they care about so they don't have to translate
    the entire point array every time they use a node. Even a huge IoT system
    has a finite # of devices that can easily fit into memory of modern
    servers/machines.
- Q: _this seems crude not to have full featured protobuf types with all the
  fields explicitly defined in protobuf. Additionally, can't protobuf handle
  type changes elegantly?_
  - A: protobuf can handle field additions and removal but we still have the
    edge cases where a point is sent from an old version of software that does
    not contain information written by a newer versions. Also, I'm not sure it
    is a good idea to have application specific type fields defined in protobuf,
    otherwise, you have a lot of work all along the communication chain to
    rebuild everything every time anything changes. With a generic types that
    rarely have to change, your core infrastructure can remain stable and any
    features only need to touch the edges of the system.
- Q: _with nodes and points, we can only represent a type with a single level of
  fields_
  - A: this is not quite true, because with the key/index fields, we can now
    have array and map fields in a node. However, the point is taken that a node
    with its points cannot represent a deeply nested data structure. However,
    nodes can be nested to represent any data structure you like. This
    limitation is by design because otherwise synchronization would be very
    difficult. By limiting the complexity of the core data structures, we are
    making synchronization and storage very simple. The tradeoff is a little
    more work to marshall/unmarshall node/point data structures into useful
    types in your application. However, marshalling code is easy compared to
    distributed systems, so we need to optimize the system for the hard parts. A
    little extra typing will not hurt anyone, and tooling could be developed if
    needed to assist in this.

Generic core data structures also opens up the possibility to dynamically extend
the system at run time without type changes. For instance, the GUI could render
new nodes it has never seen before by sending it configuration nodes with
declarative instructions on how to display the node. If core types need to
change to do this type of thing, we have no chance at this type of intelligent
functionality.

## Consequences

Removing the Min/Max/Duration fields should not have any consequences now as I
don't think we are using these fields yet.

Quite a bit of code needs to change to remove ID and add Key to code using
points.

## Additional Notes/Reference

We also took a look at how to resolve loops in the node tree:

https://github.com/simpleiot/simpleiot/issues/294

This is part of the verification to confirm our basic types are robust and have
adequate CRDT properties.
