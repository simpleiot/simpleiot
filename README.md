<img src="docs/images/simple-iot-logo.png?raw=true" width="150">

[![Go Reference](https://pkg.go.dev/badge/github.com/simpleiot/simpleiot.svg)](https://pkg.go.dev/github.com/simpleiot/simpleiot)
![Go](https://github.com/simpleiot/simpleiot/workflows/Go/badge.svg?branch=master)
![code stats](https://tokei.rs/b1/github/simpleiot/simpleiot?category=code)
[![Go Report Card](https://goreportcard.com/badge/github.com/simpleiot/simpleiot)](https://goreportcard.com/report/github.com/simpleiot/simpleiot)
[![Slack Widget](https://img.shields.io/badge/join-us%20on%20slack-gray.svg?longCache=true&logo=slack&colorB=red)](http://gophers.slack.com/messages/simpleiot)

Simple IoT is collection of building blocks to help you build custom IoT systems
quickly, but yet provide full flexibility to customize the system. Many features
such as device communication, device update, rules, user/group management, user
portal, etc. are needed for every IoT system. This project provides a solid
foundation of common features so that you can focus on the specific problem you
are solving.

## Guiding principles

1. Simple concepts are flexible and scale well.
1. There are more problems to solve than people to solve them, thus it makes
   sense to collaborate on the common technology pieces.
1. There are a lot of IoT applications that are
   [not Google](https://blog.bradfieldcs.com/you-are-not-google-84912cf44afb)
   scale (10-1000 device range).
1. There is significant opportunity in the
   [long tail](https://www.linkedin.com/pulse/long-tail-iot-param-singh) of IoT,
   which is our focus.
1. There is value in custom solutions (programming vs drag-n-drop).
1. There is value in running/owning our own platform.
1. A single engineer should be able to build and deploy a custom IoT system.
1. We don't need to spend excessive amounts of time on operations. For smaller
   deployments, we deploy one binary to a cloud server and we are done with
   operations. We don't need 20 microservices when one
   [monolith](https://m.signalvnoise.com/the-majestic-monolith/) will
   [work](https://changelog.com/posts/monoliths-are-the-future) just
   [fine](https://m.signalvnoise.com/integrated-systems-for-integrated-programmers/).
1. For many applications, a couple hours of down time is not the end of the
   world. Thus a single server that can be quickly rebuilt as needed is adequate
   and in many cases more reliable than complex systems with many moving parts.

## Core features/requirements:

1. Runs in cloud and edge instances.
1. Configuration changes can be made at either cloud, UI, or edge and are
   synchronized efficiently in any direction.
1. Efficient use of network bandwidth for edge systems as most are connected via
   low cost cellular plans.
1. Rules can run in cloud or edge devices depending on action required (sending
   notifications or controlling outputs).
1. System supports user authentication and grouping users and devices at
   multiple levels.
1. User interface updates to changes in real time.
1. Easy to add custom logic/algorithms by adding processes written in any
   language that connect to Simple IoT via NATS.

IoT Systems are inherently distributed, so even though we prefer a monolith for
a cloud service, we can't get around the fact that we also need to synchronize
data with edge devices and user interfaces.

![System topology](docs/images/distributed.png)

Simple IoT is built on simple data structures arranged in a graph that allows
for very flexible configurations.

![Nodes](docs/images/nodes2.png)

Because the core of Simple IoT is designed with flexible data structures, adding
functionality and supporting new devices is usually as simple as creating your
custom edge device code and modifying the UI to display and configure your
device features.

![What changes](docs/images/constant-vs-varying-system-parts.png)

Though we are focusing on smaller deployments initially, there is no reason
Simple IoT can't scale to large systems by swapping out the internal database
for MongoDB/Dgraph/InfluxDB/etc.

See [vision](docs/vision.md) and [architecture](docs/architecture.md) for
addition discussion on these points.

This project was developed while building real-world applications and has
primarily been driven by these project requirements. This project provides

- an stand-alone application that can be deployed at the edge or in the cloud
- [packages](https://pkg.go.dev/github.com/simpleiot/simpleiot) for implementing
  an edge application to run on embedded Linux systems.

The Simple IoT project also includes open source gateway
[firmware](https://github.com/simpleiot/firmware/tree/master/siot-fw) and
[hardware](https://github.com/simpleiot/hardware) designs.

[Detailed documentation](docs/README.md)

## Example 1

This example (build only tested on Linux and MacOS) shows how to run the server
and send data to it:

Build Simple Iot or download the latest release:

- install Go v1.14 (newer versions will likely work) and node/npm (tested with
  v12 and v14)
- git clone https://github.com/simpleiot/simpleiot.git
- `cd simpleiot`
- `. envsetup.sh` (note space is required between `.` and `envsetup.sh`. Another
  way to type this is `source envsetup.sh`. This command populates your terminal
  session with all the functions defined in `envsetup.sh`.)
- `siot_setup`
- `siot_build`

Now, run the example:

- in one terminal, start server: `./siot`
- open http://localhost:8080
  - login with user `admin@admin.com` and password `admin`
- in another terminal, send some data
  - using HTTP: `./siot -sendPoint "1823:t1:23.5:temp"`
  - using NATS: `./siot -sendPointNats "1234:v2:12.5:volt"`
  - (the format of the `-sendPoint` argument is: `devId:sensId:value:type`)
- in a few seconds, devices should be populated in the web application

### SIOT web interface screenshot

Below is a screenshot of the siot web interface with the above data.

![nodes](docs/images/screenshot-example1.png)

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

- `./siot -sendPointText "1234::1:hwVersion`
- `./siot -sendPointText "1234::2:osVersion`
- `./siot -sendPointText "1234::3:appVersion`

## Flexible node view

Information is arranged in a flexible node/tree which allows for easy grouping
of users, devices, and device attributes.

![nodes](docs/images/screenshot-nodes.png)

## Each nodes can be expanded to edit/view attributes

![node-edit](docs/images/screenshot-node-edit.png)

## Examples/tutorial vidoes

Our [YouTube channel](https://www.youtube.com/channel/UCDAtjx0utMbJCexZ7Q5CbNg)
has many more examples/tutorials of how to use Simple IoT upstream connections,
remote access to PLCs, Modbus configuration, etc.

## Dashboard and Graphing

Although Simple IoT provides a rudimentary dashboard and device listing, it does
not provide graphs yet. If you need graphs, using InfluxDb + Grafana may be a
good interim solution. [Contact](https://community.tmpdir.org/c/simple-iot/5) us
if you need help setting this up -- it is relatively simple.

## Features

Note, Simple IoT is under heavy development right now and APIs and database
format may change. If you can't find something, it likely got moved to a
different package, or renamed -- feel free to ask if you run into problems.

- [x] edit/save device config
- [x] device management
- [x] dashboard showing each device and collected parameters
- [x] REST [api](docs/API.md) for devices
- [x] [particle.io](docs/configuration.md) support
- [x] boltdb support
- [x] user authentication
- [x] user accounts
- [x] group support (assign users and devices to a group so users can only see
      devices they own).
- [x] [Modbus RTU pkg](https://pkg.go.dev/github.com/simpleiot/simpleiot/modbus)
      (both client and server)
- [x] [rules engine](docs/rules.md) (conditions/consequences)
- [x] [sms](docs/configuration.md) notifications
- [x] [modem/network management](https://pkg.go.dev/github.com/simpleiot/simpleiot/network)
- [x] NATS.io integration
- [x] efficient protocols for cellular data connections (NATs/protobuf)
- [x] Modbus (server & client) RTU support in SIOT
- [x] Modbus (server & client) TCP support in SIOT
- [x] Twilio SMS notifications
- [x] influxdb 2.x support
- [x] bi-directional synchronization with upstream instances
- [ ] file transfer over NATs (used for sw updates)
- [ ] email notifications
- [ ] COAP API for devices
- [ ] store timeseries data in bolt
- [ ] esp32 client example
- [ ] graph timeseries data
- [ ] WiFi management
- [ ] Graphs

## Support, Community, Contributing, etc.

Pull requests are welcome -- see [development](docs/DEVELOPMENT.md) for more
thoughts on architecture, tooling, etc. Issues are labelled with "help wanted"
and "good first issue" if you would like to contribute to this project.

For support or to discuss this project, use one of the following options:

- [Simple IoT community forum](https://community.tmpdir.org/c/simple-iot/5)
- #simpleiot Slack channel is available on
  [gophers.slack.com](https://gophers.slack.com/messages/simpleiot/)
- open a Github issue

Other resources:

- [Simple IoT YouTube channel](https://www.youtube.com/channel/UCDAtjx0utMbJCexZ7Q5CbNg)

## License

Apache Version 2.0
