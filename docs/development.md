---
id: development
title: Development
sidebar_label: Development
---

## Coding Standards

Please run `siot_test` from `envsetup.sh` before submitting pull requests. All
code should be formatted and linted before committing.

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
`frontend/src/Data` directory for Elm code. A few of these data structures
include:

- `Device`: represents a IoT device that is capable of communicating with a siot
  server. Typically a Device functions as a gateway or edge device aggregating
  data from a number of sensors and then sending the data to the siot server.
- `Sample`: a sample of sensor data.
- `Config`: defines a configuration parameter for a Device.

## Extendible architecture

Any `siot` app can function as a standalone, client, server or both. As an
example, `siot` can function both as an edge (client) and cloud apps (server).

- client: initiates and maintains connection with server. Can be behind a
  firewall, NAT, etc.
- server: needs to be on a network that is accessible by clients

## Configuration

(this section is WIP -- just ideas at this point)

As Simple IoT is evolving into a distributed system, the question of
configuration and the synchronization of config needs to be considered. Both
client and server siot instances can make configuration changes. A example might
be a edge device that has a local LCD and keypad that allows the user to make
configuration changes. This configuration is synchronized to a server instance
and changes can be made there as well. Both instances will need to communicate
changes to the other instance and know if they are in sync.

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
