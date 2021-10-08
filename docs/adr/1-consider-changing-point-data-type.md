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
sensor/configuration value with just a few fields. Making this more general also
increases the "evolvability" (Kleppmann) of the system. From his book:

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

The existing min/max would just become fields. This would map better into
influxdb. There would be some redundancy between Type and Field keys.

One argument against this change is that adding these maps would likely increase
the possibility of schema version conflicts between versions of software because
points are overwritten.

Some reference information on other standards:

https://github.com/eclipse/tahu/blob/master/sparkplug_b/sparkplug_b.proto

https://datatracker.ietf.org/doc/html/draft-ietf-core-senml-08#page-9

https://community.tmpdir.org/t/book-review-designing-data-intensive-applications/288/6?u=cbrake

Still not obvious yet if Index and Duration should stay, or could they be
encoded as fields/tags. Index is currently used for things like weekday
checkboxs in schedule rules.

In this case, the condition node has a series of weekday points with indexes 0-6
to represent the days of the week.

This could also be represented in the field map:

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

## Decision

## Consequences
