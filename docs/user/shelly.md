# Shelly IoT

Shelly sells a number of reasonably priced open IoT devices for home automation
and industrial control. Most support Wi-Fi network connections and some of the
Industrial line also supports Ethernet. The API is open and the devices support
a number of communication protocols including HTTP, MQTT, CoAP, etc. They also
support mDNS so they can be discovered on the network.

- [Main website](https://www.shelly.cloud/)
- [Device documentation](https://kb.shelly.cloud/knowledge-base/devices)
- [API](https://shelly-api-docs.shelly.cloud/)

Simple IoT provides the following support:

- Automatic discovery of all Shelly devices on the network using mDNS
- Support for the following devices:
  - `1pm` (not tested)
  - `Bulb Duo` (on/off only)
  - `Plus 1`
  - `Plus 1PM` (not tested)
  - `Plus 2PM`
  - `Plus Plug` (only US variant tested)
    - Measurements such as Current, Power, Temp, Voltage are collected.
  - `Plus i4`
- Currently status is polled via HTTP every 2 seconds

## Setup

- Configure the Shelly devices to connect to your Wi-Fi network. There are
  several options:
  1. Use the Shelly phone app
  1. A new device will start up in access point mode. Attach a computer or phone
     to this AP, open [http://192.168.33.1](http://192.168.33.1) (default
     address of a reset device), and then configure the Wi-Fi credentials using
     the built-in Web UI.
- Add the Shelly client in SIOT
- The Shelly client will then periodically scan for new devices and add them as
  child nodes.

## Example

![shelly](images/shelly.png)

## Plug Example

![shelly plug](images/shelly-plug.png)

## Schema

The configuration of a Shelly node and one of the devices it found:

```yaml
nodes:
  - shelly:
      description: Shelly
      disabled: 0
      children:
        - shellyIo:
            controlled: 1
            description: Bench light
            deviceID: shellyplusplugus-b0b21c12ad58
            disabled: 0
            ip: 192.168.1.42
            type: PlugUS
```

The Shelly node itself only carries a description and whether it is disabled.
Everything else follows from what it finds: the client scans the network and
adds a child node for each device, filling in `deviceID`, `ip`, and `type`
itself. `type` is the device model, such as `PlugUS`, `Plus1`, `Plus2PM`, or
`PlusI4`.

What you configure on a device is its `description`, whether it is `disabled`,
and, for a device that can be driven, `controlled`. With `controlled` set, the
client drives the device to the `switchSet` and `lightSet` values whenever they
differ from what the device reports, which is what lets a rule or the UI change
its state.

The readings and states, along with whether the device is currently reachable,
are points the client maintains, so an export of a running node carries them as
well. A device with more than one channel carries one point per channel, keyed
by channel number.
