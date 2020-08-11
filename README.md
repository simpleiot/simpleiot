<img src="docs/simple-iot-logo.png?raw=true" width="150">

![Go](https://github.com/simpleiot/simpleiot/workflows/Go/badge.svg?branch=master)

Simple IoT is collection of building blocks and best practices for building IoT
systems, learned from experience building real-world systems.

Demo is running at: https://portal.simpleiot.org/

The Simple IoT project also includes open source gateway
[firmware](https://github.com/simpleiot/firmware/tree/master/siot-fw) and
[hardware](https://github.com/simpleiot/hardware) designs.

[Detailed Documentation](docs/README.md)

## Example 1 (build from source)

This example shows how to run the server and simulator after cloning and
building from source.

- install Go v1.13 (newer versions may work) and node/npm
- git clone https://github.com/simpleiot/simpleiot.git
- `cd simpleiot`
- `. envsetup.sh`
- `siot_setup`
- `siot_build`
- in one terminal, start server: `./siot`
- open http://localhost:8080
  - login with user `admin@admin.com` and password `admin`
- in another terminal, send some data: `./siot -sendSample "1823:t1:23.5:temp"`
  - the format of the `-sendSample` argument is: `devId:sensId:value:type`

## Fetch data from Particle.io

If you have Particle devices running
[Simple IoT firmware](https://github.com/simpleiot/firmware), you can fetch the
data by exporting the following environment variable:

`PARTICLE_API_KEY=<token>`

Particle API tokens can be managed by using the `particle token` cli command.

## Store sample data in influxdb

You can store data in influxdb 1.x by exporting the following environment
variables:

- `INFLUX_URL`
- `INFLUX_USER`
- `INFLUX_PASS`
- `INFLUX_DB`

## Running unit tests

There are not a lot of unit tests in the project yet, but below are some
examples of running tests:

- test everything: `go test ./...`
- test only db directory: `go test ./db`

The leading `./` is important, otherwise Go things you are giving it a package
name, not a directory. The `...` tells Go to recursively test all subdirs.

## Vision

This section describes some high level ideas for the project and are further
described in the project [documentation](docs/README.md).

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

## Features

- [x] edit/save device config
- [x] device management
- [x] simple dashboard for each device showing collected parameters
- [x] REST [api](docs/API.md) for devices
- [x] [particle.io](docs/environment-variables.md) support
- [x] boltdb support
- [x] [influxdb 1.x](docs/environment-variables.md) support
- [x] user authentication
- [x] user accounts
- [x] group support (assign users and devices to a group so users can only see
      devices they own).
- [x] [Modbus RTU pkg](https://pkg.go.dev/github.com/simpleiot/simpleiot/modbus)
      (both client and server)
- [x] Command line Modbus utlity
- [x] [rules engine](docs/rules.md) (conditions/consequences)
- [x] [sms](docs/environment-variables.md) notifications
- [ ] NATS.io integration
      ([WIP](https://github.com/simpleiot/simpleiot/tree/feature-nats))
- [ ] file transfer API (over NATs)
- [ ] email notifications
- [ ] COAP API for devices
- [ ] influxdb 2.x support
- [ ] store timeseries data in bolthold
- [ ] esp32 client example
- [ ] graph timeseries data
- [ ] efficient protocols for cellular data connections (CoAP, protobuf, etc.)
- [ ] Google Cloud Datastore
- [ ] App Engine Deployment
- [ ] edge computing features (rules, etc)

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

## Pull Requests Welcome

This is a community project. See [development](docs/DEVELOPMENT.md) for more
thoughts on architecture, tooling, etc.

For support or to discuss this project, please visit the
[Simple IoT community forum](https://community.tmpdir.org/c/simple-iot/5)

## License

Apache Version 2.0
