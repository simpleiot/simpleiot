<img src="docs/simple-iot-logo.png?raw=true" width="150">

![Go](https://github.com/simpleiot/simpleiot/workflows/Go/badge.svg?branch=master)

Simple IoT is collection of building blocks to help you build custom IoT systems
quickly, but yet provide full flexibility to customize the system. Many features
such as device communication, device update, rules, user/group management, user
portal, etc. are needed for every IoT system. This project provides a solid
foundation of common features so that you can focus on the specific problem you
are solving.

A few guiding principles.

1. Simple concepts are flexible and scale well.
1. There are more problems to solve than people to solve them, thus it makes
   sense to collaborate on the common technology pieces.
1. There are a lot of IoT applications that are
   [not Google](https://blog.bradfieldcs.com/you-are-not-google-84912cf44afb)
   scale (10-1000 device range).
1. There is significant opportunity in the
   [long tail](https://www.linkedin.com/pulse/long-tail-iot-param-singh) of IoT,
   which is our focus. This is not an "enterprise" platform.
1. There is value in custom solutions (programming vs drag-n-drop).
1. There is value in running our own platform.
1. A single engineer should be able to build and deploy a custom IoT system.
1. We don't need to spend gobs of time on operations. For smaller deployments,
   we deploy one binary to a cloud server and we are done with operations. We
   don't need 20 microservices when one
   [monolith](https://m.signalvnoise.com/the-majestic-monolith/) will
   [work](https://changelog.com/posts/monoliths-are-the-future) just
   [fine](https://m.signalvnoise.com/integrated-systems-for-integrated-programmers/).
1. For many applications, a couple hours of down time is not the end of the
   world. Thus a single server that can be quickly rebuilt as needed is
   adequate.

Though we are focusing on smaller deployments initially, there is no reason
Simple IoT can't scale to large systems by swapping out the internal database
for MongoDB/Dgraph/InfluxDB/etc.

See [vision](docs/vision.md) for addition discussion.

This project was developed while building real-world applications and has
primarily been driven by these project requirements. This project provides

- a portal application (typically deployed in the cloud)
- [packages](https://pkg.go.dev/github.com/simpleiot/simpleiot) for implementing
  an edge application to run on embedded Linux systems.

The Simple IoT project also includes open source gateway
[firmware](https://github.com/simpleiot/firmware/tree/master/siot-fw) and
[hardware](https://github.com/simpleiot/hardware) designs.

[Detailed documentation](docs/README.md)

## Example 1 (build from source)

This example (only tested on Linux and MacOS) shows how to run the server and
send data to it:

- install Go v1.14 (newer versions will likely work) and node/npm (tested with
  v12 and v14)
- git clone https://github.com/simpleiot/simpleiot.git
- `cd simpleiot`
- `. envsetup.sh` (note space is required between `.` and `envsetup.sh`. Another
  way to type this is `source envsetup.sh`. This command populates your terminal
  session with all the functions defined in `envsetup.sh`.)
- `siot_setup`
- `siot_build`
- in one terminal, start server: `./siot`
- open http://localhost:8080
  - login with user `admin@admin.com` and password `admin`
- in another terminal, send some data
  - using HTTP: `./siot -sendPoint "1823:t1:23.5:temp"`
  - using NATS: `./siot -sendPointNats "1234:v2:12.5:volt"`
  - (the format of the `-sendPoint` argument is: `devId:sensId:value:type`)
- in a few seconds, devices should be populated in the web application

### SIOT web interface screenshot

Below is a screenshot of the siot web interface. The greyed out device is one
that is longer communicating and is considered "offline".

![portal](docs/portal-devices.png)

## Example 2 (send commands/files to device)

- `./siot`
- in another terminal, start edge device example: `go run cmd/edge/main.go`
- in a 3rd terminal:
  - send command to device: `./siot -sendCmd=setTank:150`
  - send file to device:
    `./siot -sendFile=https://raw.githubusercontent.com/simpleiot/simpleiot/master/README.md`

## Example 3 (send data with acknowledgments from server)

- `./siot -sendPointNats "1234:v2:12.5:volt" -natsAck`

## Example 4 (send version information to server)

Hardware version information is a `Point` that encodes the version information
in the `Text` field of a `Point`.

- `./siot -sendPointText "1234::1:hwVersion"
- `./siot -sendPointText "1234::2:osVersion"
- `./siot -sendPointText "1234::3:appVersion"

## User/Group management

Users can be added to the system. If a user and device are in the same group,
then the user can see the device.

### User page screenshot

![users](docs/portal-users.png)

### Group page screenshot

In the below example, Joe and Cliff will both have access to the "1234" device
because they are all in the "Building A" group.

![groups](docs/portal-groups.png)

## Configuration

Simple IoT can be [configured](docs/configuration.md) to connect with a number
of external programs/services such as Particle.io, Twilio, and Influxdb.

Additionally, command line option help can be viewed by running `siot --help`.

## Dashboard and Graphing

Although Simple IoT provides a rudimentary dashboard and device listing, it does
not provide graphs yet. If you need graphs, using InfluxDb + Grafana may be a
good interim solution. [Contact](https://community.tmpdir.org/c/simple-iot/5) us
if you need help setting this up -- it is relatively simple.

## Features

Note, Simple IoT is under heavy development right now and APIs may change. If
you can't find something, it likely got moved to a different package, or renamed
-- feel free to ask if you run into problems.

- [x] edit/save device config
- [x] device management
- [x] dashboard showing each device and collected parameters
- [x] REST [api](docs/API.md) for devices
- [x] [particle.io](docs/configuration.md) support
- [x] boltdb support
- [x] [influxdb 1.x](docs/configuration.md) support
- [x] user authentication
- [x] user accounts
- [x] group support (assign users and devices to a group so users can only see
      devices they own).
- [x] [Modbus RTU pkg](https://pkg.go.dev/github.com/simpleiot/simpleiot/modbus)
      (both client and server)
- [x] Command line Modbus utlity
- [x] [rules engine](docs/rules.md) (conditions/consequences)
- [x] [sms](docs/configuration.md) notifications
- [x] [modem/network management](https://pkg.go.dev/github.com/simpleiot/simpleiot/network)
- [x] NATS.io integration
      ([WIP](https://github.com/simpleiot/simpleiot/tree/feature-nats))
- [x] file transfer over NATs (used for sw updates)
- [x] efficient protocols for cellular data connections (NATs/protobuf)
- [ ] email notifications
- [ ] COAP API for devices
- [ ] influxdb 2.x support
- [ ] store timeseries data in bolthold
- [ ] esp32 client example
- [ ] graph timeseries data
- [ ] WiFi management
- [ ] Graphs

## Support, Contributing, etc.

Pull requests are welcome -- see [development](docs/DEVELOPMENT.md) for more
thoughts on architecture, tooling, etc. Issues are labelled with "help wanted"
and "good first issue" if you would like to contribute to this project.

For support or to discuss this project, use one of the following options:

- [Simple IoT community forum](https://community.tmpdir.org/c/simple-iot/5)
- #simpleiot Slack channel is available on
  [gophers.slack.com](https://gophers.slack.com/messages/simpleiot/)
- open a github issue

## License

Apache Version 2.0
