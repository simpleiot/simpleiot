# Installation

Simple IoT will run on the following systems:

- ARM/x86/RiscV Linux
- MacOS
- Windows

The computer you are currently using is a good platform to start with as well as
any common embedded Linux platform like the Raspberry PI or Beaglebone Black.

If you needed an industrial class device, consider something from embeddedTS
like the [TS-7553-V2](https://www.embeddedts.com/products/TS-7553-V2).

The Simple IoT application is a self contained binary with no dependencies.
Download the [latest release](https://github.com/simpleiot/simpleiot/releases)
for your platform and run the executable. Once running, you can log into the
user interface by opening [http://localhost:8118](http://localhost:8118) in a
browser. The default login is:

- user: `admin@admin.com`
- pass: `admin`

### Simple IoT self-install (Linux only)

Simple IoT self-installation does the following:

- creates a systemd service file
- creates a data directory
- starts and enables the service

To install as user, run:

`siot install`

To install as root:

`sudo siot install`

The default ports are used, so if you want something different, modify the
generated `siot.service` file.

## Cloud/Server deployments

When on the public Internet, Simple IoT should be proxied by a web server like
Caddy to provide TLS/HTTPS security. Caddy by default obtains free TLS
certificates from Let's Encrypt and ZeroSSL with automatic fallback if one
provider fails.

There are Ansible recipes available to deploy Simple IoT, Caddy, Influxdb, and
Grafana that work on most Linux servers.

- [Simple IoT](https://github.com/simpleiot/ansible-role-simpleiot-bin)
- [Caddy, Influxdb, Grafana, etc](https://github.com/cbrake?tab=repositories&q=ansible)

### [Video: Setting up a Simple IoT System in the cloud](https://youtu.be/pH8GPbjt-SI)

<iframe width="791" height="445" src="https://www.youtube.com/embed/pH8GPbjt-SI" title="Setting up a Simple IoT System in the cloud" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" allowfullscreen></iframe>

## Yocto Linux

Yocto Linux is a popular edge Linux solution. There is a
[Bitbake recipe](https://github.com/YoeDistro/yoe-distro/blob/master/sources/meta-yoe/recipes-siot/simpleiot/simpleiot_git.bb)
for including Simple IoT in Yocto builds.

## Networking

By default, Simple IoT runs an embedded NATS server and the SIOT NATS client is
configured to connect to `nats://127.0.0.1:4222`.
