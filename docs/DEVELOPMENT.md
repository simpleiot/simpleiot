---
id: development
title: Development
sidebar_label: Development
---

This document attempts to outlines the basic architecture and development
philosophy. The basics are covered in the [readme](../README.md). As the name
suggests, a core value of the project is simplicity. Thus any changes should be
made with this in mind. Although this project has already proven useful on
several real-world project, it is a work in progress and will continue to
improve.

## Technology choices

Choices for the technology stack emphasize simplicity, not only in the language,
but just as important, in the deployment and tooling.

- **Backend**
  - [Go](https://golang.org/)
    - simple language and deployment model
    - nice balance of safety + productivity
    - excellent tooling and build system
- **Frontend**
  - Single Page Application (SPA) architecture
    - programming environment is much more powerful than server rendered pages
      (PHP, Rails, etc).
    - easier to transition to Progressive Web Apps (PWA)
  - [Elm](https://elm-lang.org/)
    - nice balance of safety + productivity
    - excellent compiler messages
    - reduces possibility for run time exceptions in browser
    - does not require a huge/complicated/fragile build system typical in
      Javascript frontends.
  - [elm-ui](https://github.com/mdgriffith/elm-ui)
    - What if you never had to write CSS again?
    - a fun, yet powerful way to lay out a user interface and allows you to
      efficiently make changes and get the layout you want.
- **Database**
  - Eventually support multiple databased backends depending on scaling/admin
    needs
  - Embedded db using [BoltHold](https://github.com/timshannon/bolthold)
    - no external services to configure/admin
- **Hosting**
  - Any server that provides ability run long-lived Go applications (Digital
    Ocean, Linode, GCP compute engine, AWS ec2, etc)

In our experience, simplicity and good tooling matter. It is easy to add
features to a language, but creating a useful language/tooling that is simple is
hard. Since we are using Elm on the frontend, it might seem appropriate to
select a functional language like Elixir, Scala, Clojure, etc. for the backend.
These environments are likely excellent for many projects, but are also
considerably more complex to work in. The programming style (procedural,
functional, etc.) is important, but other factors such as
simplicity/tooling/deployment are also important, especially for small teams who
don't have separate staff for backend/frontend/operations. Learning two simple
languages (Go and Elm) is a small task compared to dealing with huge languages,
fussy build tools, and complex deployment environments.

This is just a snapshot in time -- there will likely be other better technology
choices in the future. The backend and frontend are independent. If either needs
to be swapped out for a better technology in the future, that is possible.

## Vision

This section describes some high level ideas for the project. Much of this is
just at the brainstorming stage and has not been implemented yet.

- built around collecting and visualizing data from devices
- provide a good base system to build IoT products that may support a number of
  devices, users, rules, etc.
- is useful out of the box, but requires typically requires coding to customize
  for specific applications. This is not a GUI for building IoT systems, but
  rather a code base software developers can use as a starting point.
- easy to extend for new devices or custom applications.
- the `siot` app can be a client or server. Any `siot` app can be a stand-alone
  IoT system or act as a client and forward data to another `siot` instance.
  Consider this example:
  1. run `siot` app on rPI to collect data from sensors attached to it. Web UI
     can be accessed at the rPI IP address.
  1. the rPI `siot` instance forwards data to another `siot` instance running on
     a server in your local network.
  1. the server `siot` instance forwards data to another `siot` instance in the
     cloud.
- data can be synchronized in any direction, as long as the receiving device is
  on an accessible network. Sending devices always initiate the connection, and
  can thus be behind a firewall or NAT. Typically an edge gateway collects data
  from sensors and sends it to a cloud server. But you could also have two cloud
  servers that send data to each other if they are both configured as upstream
  instances.
- configuration can be synchronized between clients and servers in either
  direction.
- application technology is general, so you are not dependant on any one IoT
  company or cloud provider
- plugin architecture for extending the system with custom functionality
- easy to set up for small/mid size deployments -- not a lot of moving parts to
  worry about. Can be deployed in-house if you don't need data in the cloud.
- simple deployment process (not a lot of pieces on the backend to manage)
  - Deployment/upgrade is as simple as copying one binary.
  - All assets are embedded.
  - For small deployments (< 1000 devices), application is completely self
    contained (no external databases or services are required).
- Storage (config and sensor data) supports multiple databases
  - embedded db for small deployments
  - (mongodb, Google Cloud Datastore, influxdb, etc) for larger deployments or
    other needs.

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
