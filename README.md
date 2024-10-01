<img src="docs/images/simple-iot-logo.png?raw=true" width="150">

[![Go Reference](https://pkg.go.dev/badge/github.com/simpleiot/simpleiot.svg)](https://pkg.go.dev/github.com/simpleiot/simpleiot)
![Go](https://github.com/simpleiot/simpleiot/workflows/Go/badge.svg?branch=master)
[![Scc Count Badge](https://sloc.xyz/github/simpleiot/simpleiot/?category=code)](https://github.com/simpleiot/simpleiot/)
[![Go Report Card](https://goreportcard.com/badge/github.com/simpleiot/simpleiot)](https://goreportcard.com/report/github.com/simpleiot/simpleiot)
[![Slack Widget](https://img.shields.io/badge/join-us%20on%20slack-gray.svg?longCache=true&logo=slack&colorB=red)](http://gophers.slack.com/messages/simpleiot)

**Simple Iot enables you to add remote sensor data, telemetry, configuration,
and device management to your project or product.**

Implementing IoT systems is hard. Most projects take way longer and cost more
than they should. The fundamental problem is getting data from remote locations
(edge) to a place where users can access it (cloud). We also need to update data
and configuration at the edge in real time from any location. Simple IoT is an
attempt to solve these problems by embracing the fact that IoT systems are
inherently distributed and building on simple concepts that scale.

**Simple IoT** provides:

- a single application with no dependencies that can be run in both cloud and
  edge instances
- efficient synchronization of data in both directions
- a flexible UI to view configuration and current values
- a rules engine that runs on all instances that can trigger notifications or
  set data
- extensive support for Modbus -- both server and client
- support for the Linux 1-wire subsystem.
- flexible graph organization of instances, users, groups, rules, and
  configuration
- integration with other services like InfluxDB and Twilio
- a system that is easy to extend in any language using NATS
- a number of useful Go packages to use in your custom application

See [vision](docs/ref/vision.md), [architecture](docs/ref/architecture.md), and
[integration](docs/ref/integration.md) for addition discussion on these points.

See [detailed documentation](https://docs.simpleiot.org) for installation,
usage, and development information.

## Motivation

This project was developed while building real-world IoT applications and has
been driven by the following requirements:

- Data (state or configuration) can be changed anywhere — at edge devices or in
  the cloud and this data needs to be synchronized seamlessly between instances.
  Sensors, users, rules, etc. can all change data. Some edge systems have a
  local display where users can modify the configuration locally as well as in
  the cloud. Rules can also run in the cloud or on edge devices and modify data.
- Data bandwidth is limited in some IoT systems — especially those connected
  with Cat-M modems (< 100kb/s). Additionally, connectivity is not always
  reliable, and systems need to continue operating if not connected.

## Core ideas

The process of developing Simple IoT has been a path of reducing what started as
a fairly complex IoT system to simpler ideas. This is what we discovered along
the way:

1. treat configuration and state data the same for purposes of storage and
   synchronization.
2. represent this data using simple types (Nodes and Points).
3. organize this data in a graph.
4. all data flows through a message bus.
5. run the same application in the cloud and at the edge.
6. automatically sync common data between instances.

> [Design is the beauty of turning constraints into advantages.](https://www.webstock.org.nz/talks/design-is-the-beauty-of-turning-constraints-into-advantages/)
> -- Ava Raskin

These constraints have resulted in Simple IoT becoming a flexible distributed
graph database optimized for IoT datasets. We'll explore these ideas more in the
[documentation](https://docs.simpleiot.org).

## Support, Community, Contributing, etc.

Pull requests are welcome -- see [development](docs/ref/development.md) for more
thoughts on architecture, tooling, etc. Issues are labelled with "help wanted"
and "good first issue" if you would like to contribute to this project.

For support or to discuss this project, use one of the following options:

- [Documentation](https://docs.simpleiot.org)
- [Simple IoT community forum](https://community.tmpdir.org/c/simple-iot/5)
- #simpleiot Slack channel is available on
  [gophers.slack.com](https://gophers.slack.com/messages/simpleiot/)
- open a Github issue
- [Simple IoT YouTube channel](https://www.youtube.com/@simple-iot)
- subscribe to our [email newsletter](https://tmpdir.ck.page/196d1fb480) for
  project updates.

## License

Apache Version 2.0

## Contributors

Thanks to contributors:

<a href="https://github.com/simpleiot/simpleiot/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=simpleiot/simpleiot" />
</a>

Made with [contrib.rocks](https://contrib.rocks).
