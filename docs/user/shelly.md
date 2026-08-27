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
- Support for any Gen2 or later device, including ones released after this was
  written. A device reports its own components, so what Simple IoT reads from it
  follows from what it has rather than from a list of models.
- Support for Gen1 devices through their HTTP API
- Status arrives by push on Gen2 and later, over a WebSocket the device sends
  updates on as they happen. Gen1 devices are polled every 2 seconds.

## How a device is read

A Gen2 or later device answers `Shelly.GetStatus` with its whole state, keyed by
component: `switch:0`, `input:1`, `cover:0`, `em:0`, `temperature:100`. Simple
IoT reads that list and creates one point per component, keyed by the component
id. A device with two relays reports two switches; an add-on module contributing
a temperature sensor reports it alongside the rest.

Measurements follow the same rule. A switch that reports `apower`, `voltage`,
and `current` has power monitoring, so those points appear; a switch without it
reports none of them and none appear. Nothing in Simple IoT records which models
measure power.

Simple IoT keeps a WebSocket open to each Gen2 or later device. Once connected,
the device pushes each change as it happens, so a relay switched at the wall
shows up right away rather than at the next poll. Simple IoT still reads the
whole device once a minute as a backstop, and treats loss of the connection as
the device going offline.

The following point types can appear, depending on the device: `switch`,
`light`, `input`, `position`, `coverState`, `power`, `voltage`, `current`,
`energy`, `powerFactor`, `apparentPower`, `frequency`, `temp`, `humidity`,
`brightness`, `white`, `lightTemp`, `battery`, `batteryLevel`, `externalPower`,
and `alarm`.

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
            gen: 2
            ip: 192.168.1.42
            type: SNPL-00116US
```

The Shelly node itself only carries a description and whether it is disabled.
Everything else follows from what it finds: the client scans the network and
adds a child node for each device, filling in `deviceID`, `ip`, `gen`, and
`type` itself. `type` is the model the device reports for itself, such as
`SNPL-00116US` for a Plus Plug US or `SHPLG-S` for a Gen1 Plug S.

What you configure on a device is its `description`, whether it is `disabled`,
and, for a device that can be driven, `controlled`. With `controlled` set, the
client drives the device to the `switchSet`, `lightSet`, and `positionSet`
values whenever they differ from what the device reports, which is what lets a
rule or the UI change its state.

The readings and states, along with whether the device is currently reachable,
are points the client maintains, so an export of a running node carries them as
well. A device with more than one channel carries one point per channel, keyed
by the component id the device uses. Those ids are not always a dense range: an
add-on module numbers its components from 100, so a Plus 1 with an add-on
carries `switch` and `input` keyed `0` alongside `temperature` keyed `100`.
