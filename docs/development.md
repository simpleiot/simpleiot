---
id: development
title: Development
sidebar_label: Development
---

## Code Organization

Currently, there are a lot of subdirectories. One reason for this is to limit
the size of application binaries when building edge/embedded Linux binaries. In
some use cases, we want to deploy app updates over cellular networks, therefore
we want to keep packages as small as possible. For instance, if we put the
`natsserver` stuff in the `nats` package, then app binaries grow a couple MB,
even if you don't start a nats server. It is not clear yet what Go does for dead
code elimination, but at this point, it seems referencing a package increases
the binary size, even if you don't use anything in it. (Clarification welcome!)

For edge applications on Embedded Linux, we'd eventually like to get rid of
net/http, since we can do all network communications over NATS. We're not there
yet, but be careful about pulling in dependencies that require net/http into the
nats package, and other low level packages intended for use on devices.

## Coding Standards

Please run `siot_test` from `envsetup.sh` before submitting pull requests. All
code should be formatted and linted before committing.

Please configure your editor to run code formatters:

- Go: `goimports`
- Elm: `elm-format`
- Markdown: `prettier` (note, there is a `.prettierrc` in this project that
  configures prettier to wrap markdown to 80 characters. Whether to wrap
  markdown or not is debatable, as wrapping can make diff's harder to read, but
  Markdown is much more pleasant to read in an editor if it is wrapped. Since
  more people will be reading documentation than reviewing, lets optimize for
  the reading in all scenarios -- editor, Github, and generated docs)

* [Environment Variables](environment-variables.md)

## Running unit tests

There are not a lot of unit tests in the project yet, but below are some
examples of running tests:

- test everything: `go test ./...`
- test only db directory: `go test ./db`

The leading `./` is important, otherwise Go things you are giving it a package
name, not a directory. The `...` tells Go to recursively test all subdirs.

## Flexible data structures

As we work on IoT systems, data structures (types) tend to emerge. Common data
structures allow us to develop common algorithms and mechanism to process data.
Instead of defining a new data type for each type of sensor, define one type
that will work with all sensors. Then it is trivial to support new sensors and
applications.

These are currently defined in the `data` directory for Go code, and
`frontend/src/Data` directory for Elm code. The fundamental data structures for
the system are `Devices` and `Points`. A `Device` can have one or more `Points`.
A `Point` can represent a sensor value, or a configuration parameter for the
device. With sensor values and configuration represented as `Points`, it becomes
easy to use both sensor data and configuration in rule or equations because the
mechanism to use both is the same. Additionally, if all `Point` changes are
recorded in a time series database (for instance Influxdb), you automatically
have a record of all configuration changes for a device.

`devices` can have parents or children and thus be represented in a hiearchy. To
add structure to the system, you simply add nested `Devices`. The `Device`
hiearchy can reresent the physical structure of the system, or it could also
contain virtual `Devices`. These virtual devices could contain logic to process
data from sensors. Several examples of virtual devices:

- a pump `Device` that converts motor current readings into pump events.
- implement moving averages, scaling, etc on sensor data.
- combine data from multiple sensors
- implement custom logic for a particular application

Eventually, it seems logical to have a scriping language where formulas can be
written to operate on any `Device:Point` data. While there are likely many other
systems that have this type of functionality (for instance Node-RED), the focus
of SimpleIoT is not for one-off systems where every device is manually
configured, but rather for a system that can be programmed or configured once,
and then scales with no manual effort as additional devices and users are added.

As this is a distributed system where devices may be created on any number of
connected systems, device IDs need to be unique. A unique serial number or UUID
is recommended.

When a `Point` changes, all `Devices` that depend on this data need to be
updated. This can be done in one of two ways:

- polling: simply recompute all virtual points every X amount of time. This does
  not scale.
- event driven: when a point changes, all `devices` that depend on this value
  recompute their values/rules, etc.

For the event driven model, we need to track all `Devices` that depend on a
`Point`. The simplest thing seems for the `Point` data structure to contain a
list of `Devices` that depend on the `Point`.

## Configuration and Synchronization

Typically, configuration is modifed through a user interface. As mentioned
above, the configuration of a `Device` will be stored as `Points`. Typically the
UI for a device will present fields for the needed configuration based on the
`Device:Type`.

As Simple IoT is evolving into a distributed system, the question of
configuration and the synchronization of config needs to be considered. Both
client (edge) and server (cloud) siot instances can make configuration changes.
An example might be a edge device that has a local LCD/keypad that allows the
user to make configuration changes in the field. Both client and server will
need to communicate changes to the other instance and know if they are in sync.
`Point` changes can be communicated as they are changed via NATS which both the
cloud and device instance can listen to. If one of the systems is not connected,
they will miss the `Point` change. For sensor data it is not huge deal if a
sensor reading is lost -- at some point in the future another sensor reading
will be sent. But for configuration data, it may never be changed again and it
is import that any configuration changes be synchronized. When a system comes
online (say an edge device), it requests the `Device:Points` data for all
devices it is interested in. All systems then respond with their `Point` data.
If the timestamp of a `Point` coming in is newer than the one stored locally, it
is then processed on that system. This ensures the latest information for all
`Points` is propogated (even sensor data) and should cover most scenarios even
where two people edit two different configuration parameters on the same device
on two different systems and these systems later reconnect. It may not be
appropriate for cloud systems to request `Points` for all connected `Devices` if
the service in the cloud is simply restarted (say during a new version
deployment as downtime is minimal) as then it would have a flood of `Points` to
deal with. On the other hand, NATS can handle millions of messages per second,
so with small medium/scale systems (1000's of IoT devices), it seems this is
probably not a big deal. It may also make sense to occasionally request
`Device:Points` synchronization -- say once per hour. Perhaps the `Device` data
structure could have a `LastSychronized` field and the server could initiate a
sycnronization at some interval.

TODO, how to sync device hiearchy ...

## Extendible architecture

Any `siot` app can function as a standalone, client, server or both. As an
example, `siot` can function both as an edge (client) and cloud apps (server).

- client: initiates and maintains connection with server. Can be behind a
  firewall, NAT, etc.
- server: needs to be on a network that is accessible by clients

## Frontend architecture

Much of the frontend architecture is already defined by the Elm architecture.
However, we still have to decide how data flows between various modules in the
frontend. If possible, we'd like to keep the UI
[optimistic](https://blog.meteor.com/optimistic-ui-with-meteor-67b5a78c3fcf) if
possible. Thoughts on how to accomplish this:

- single data model at top level
- modifications to the backend database are sent to the top level, the model is
  modified first, and then a request is sent to the backend to modify the
  database. This ensures the value does not flash or revert to old value while
  the backend request is being made.

## Backend architecture

Currently the backend architecture is very simple as everything is driven by
REST APIs. Eventually, we'll need to have goroutines running collecting data,
running rules, etc. and figure out how to flow data through the system.
