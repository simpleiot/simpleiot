<img src="docs/simple-iot-logo.png?raw=true" width="150">

![Go](https://github.com/simpleiot/simpleiot/workflows/Go/badge.svg?branch=master)

Simple IoT is collection of building blocks and best practices for building IoT
systems, learned from experience building real-world systems. This project
provides a portal application (typically deployed in the cloud) as well as
packages for implementing an edge application to run on embedded Linux systems.

The Simple IoT project also includes open source gateway
[firmware](https://github.com/simpleiot/firmware/tree/master/siot-fw) and
[hardware](https://github.com/simpleiot/hardware) designs.

[Detailed documentation](docs/README.md)

## Example 1 (build from source)

This example shows how to run the server and and send data to it:

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

## Configuration

Simple IoT can be [configured](docs/environment-variables.md) to connect with a
number of external programs/services such as Particle.io, Twilio, and Influxdb.

Additionally, command line option help can be viewed by running `siot --help`.

## Dashboard and Graphing

Although Simple IoT provides a rudamentary dashboard and device listing, it does
not provide graphs yet. If you need graphs, using InfluxDb + Grafana may be a
good interim solution. [Contact](https://community.tmpdir.org/c/simple-iot/5) if
you need help setting this up -- it is relatively simple.

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
- [x] [modem/network management](https://pkg.go.dev/github.com/simpleiot/simpleiot/network)
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
- [ ] WiFi management
- [ ] Graphs

## Support, Pull Requests, etc.

This is a community project. See [development](docs/DEVELOPMENT.md) for more
thoughts on architecture, tooling, etc. Issues are labelled with "help wanted"
and "good first issue" if you would like to contribute to this project.

For support or to discuss this project, please visit the
[Simple IoT community forum](https://community.tmpdir.org/c/simple-iot/5)

## License

Apache Version 2.0
