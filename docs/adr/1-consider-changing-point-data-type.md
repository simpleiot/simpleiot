# Point Data Type Changes

- Author: Cliff Brake Last updated: 2021-10-08
- Issue at: https://github.com/simpleiot/simpleiot/issues/254
- PR/Discussion: https://github.com/simpleiot/simpleiot/pull/279
- Status: Brainstorming

# Problem

The current point data type is fairly simple, but may benefit from additional or
changed fields to support more scenarios.

# Context/Discussion

Should we consider making the point struct more flexible?

The reason for this is that it is sometimes hard to describe a
sensor/configuration value with just a few fields.

# evolvability

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
> model objects in the application, and later reencode those model objects, the
> unknown field might be lost in that translation process. Solving this is not a
> hard problem; you just need to be aware of it.

Some discussion of this book:
https://community.tmpdir.org/t/book-review-designing-data-intensive-applications/288/6

One argument against this change is that adding these maps would likely increase
the possibility of schema version conflicts between versions of software because
points are overwritten.

### Other Standards

Some reference information on other standards:

https://github.com/eclipse/tahu/blob/master/sparkplug_b/sparkplug_b.proto

https://datatracker.ietf.org/doc/html/draft-ietf-core-senml-08#page-9

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
together. Then the question is how to represent the filesystem?

### Representing arrays

With the `index` field, we can already represent arrays as a group of points,
where index defines the position in the array. One example where we do this is
for selecting days of the week in schedule rule conditions. The index field is
used to select the weekday. So we can have a series of points to represent
Weekdays. In the below, Sunday is the 1st point set to 0, and Monday is the 2nd
point, set to 1.

```go
[]data.Point{
  {
    Type: "weekday",
    Index: 0,
    Value: 0,
  },
  {
    Type: "weekday",
    Index: 1,
    Value: 0,
  },
}
```

In this case, the condition node has a series of weekday points with indexes 0-6
to represent the days of the week.

If we change value to map of key/values, then weekday values could be
represented in the field map:

- "0":0
- "1":1
- "2":0
- "3":0

or

- "Sun":0
- "Mon":1
- "Tues:0
- "Wed":0

In practice, I've found presenting weekdays by numbers is easier to deal with in
programs.

A single point could then represent weekdays instead of requiring multiple
points.

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

### Schema changes and distributed synchronization

With the current point scheme, it is very easy to synchronize data, even if
there are schema changes. All points are synchronized, so one version can write
one set of points, and another version another, and all points will be sync'd to
all instances. However, if we

There is also a concern that if two different versions of the software use
different combinations of field/value keys, there could be information lost. The
simplicity and ease of merging Points into nodes is no longer simple.

## Design

The point data structure would change from:

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

to:

```go
type Point struct {
    Time time.Time
    Type string
    Index int
    Tags map[string]string
    Fields map[string]float64
    FieldsText map[string]string
}
```

The existing min/max would just become fields. This would map better into
influxdb. There would be some redundancy between Type and Field keys.

## Decision

## Consequences
