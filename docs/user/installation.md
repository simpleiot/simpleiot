# Installation

Simple IoT will run on the following systems:

- ARM/x86/RISC-V Linux
- MacOS
- Windows

The computer you are currently using is a good platform to start with as well as
any common embedded Linux platform like the Raspberry PI.

If you needed an industrial class device, consider something from embeddedTS
like the [`TS-7553-V2`](https://www.embeddedts.com/products/TS-7553-V2).

The Simple IoT application is a self contained binary with no dependencies.
Download the [latest release](https://github.com/simpleiot/simpleiot/releases)
for your platform and run the executable. On Linux and MacOS, the download needs
to be marked executable first:

```sh
chmod +x simpleiot-vX.Y.Z-linux-x86_64
./simpleiot-vX.Y.Z-linux-x86_64
```

Renaming it to `siot` is convenient if you plan to keep it in your `PATH`.

Once running, you can log into the user interface by opening
[http://localhost:8118](http://localhost:8118) in a browser. The default login
is:

- user: `admin`
- pass: `admin`

### Simple IoT self-install (Linux only)

Simple IoT self-installation does the following:

- creates a Systemd service file
- creates a data directory
- starts and enables the service

To install as user, copy the `siot` binary to some location like
`/usr/local/bin` and then run:

`siot install`

To install as root:

`sudo siot install`

The default ports are used, so if you want something different, modify the
generated `siot.service` file.

## Updating

Simple IoT can update itself to the latest release:

```sh
siot update
```

This downloads the release for the platform it is running on, verifies it
against the checksums published with the release, and replaces the binary in
place. The new binary is written to the directory the current one lives in, so
if Simple IoT is installed somewhere like `/usr/local/bin`, run
`sudo siot update`. To see what is available without installing it, use:

```sh
siot update -check
```

The new version starts running the next time Simple IoT starts, so if it is
installed as a service, restart the service:

```sh
systemctl restart siot
```

Updating replaces the executable and leaves the data directory alone, so
configuration and historical data carry forward. The previous binary is removed
once the new one is in place, so keep a copy if you want to be able to return to
it. On Windows, the previous version is left alongside the new one as
`siot.exe.old`.

Note that `siot update` updates the Simple IoT application itself. To update the
operating system on an embedded device, see the [update client](update.md).

## Cloud/Server deployments

When on the public Internet, Simple IoT should be proxied by a web server like
Caddy to provide TLS/HTTPS security. Caddy by default obtains free TLS
certificates from Let's Encrypt and ZeroSSL with automatic fallback if one
provider fails.

There are Ansible recipes available to deploy Simple IoT, Caddy, InfluxDB, and
Grafana that work on most Linux servers.

- [Simple IoT](https://github.com/simpleiot/ansible-role-simpleiot-bin)
- [Caddy, InfluxDB, Grafana, etc.](https://github.com/cbrake?tab=repositories&q=ansible)

### [Video: Setting up a Simple IoT System in the cloud](https://youtu.be/pH8GPbjt-SI)

<iframe width="791" height="445" src="https://www.youtube.com/embed/pH8GPbjt-SI" title="Setting up a Simple IoT System in the cloud" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" allowfullscreen></iframe>

## Building images

An image that runs Simple IoT on many units needs each unit to end up with its
own identity and credential. Ship a provisioning file (see
[provisioning](configuration.md#configuration-provisioning)) with a sync node
that carries an `enrollToken` and no `authToken`; each unit generates its own
key on first start and [enrolls itself](sync.md#devices-that-enroll-themselves)
with the upstream. Do not ship `device.nkey` in a shared image, since every unit
would then be the same device.

## Yocto Linux

Yocto Linux is a popular edge Linux solution. There is a
[BitBake recipe](https://github.com/openembedded/meta-openembedded/tree/master/meta-oe/recipes-extended/simpleiot)
for including Simple IoT in Yocto builds.

## Networking

By default, Simple IoT runs an embedded NATS server and the SIOT NATS client is
configured to connect to `nats://127.0.0.1:4222`.
