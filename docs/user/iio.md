# IIO Client

The IIO client reads analog values through the Linux
[Industrial I/O subsystem](https://www.kernel.org/doc/html/latest/driver-api/iio/index.html),
which is the kernel's framework for ADCs, DACs, and the sensors that behave like
them. A 4-20 mA loop through an ADS1115, a pressure sensor on an I2C bus, a
thermocouple through a MAX31856, and the accelerometer already sitting on a
gateway's board all reach Simple IoT the same way.

Each channel publishes a `value` point, which works with everything downstream:
a rule condition can watch it, a rule action can write `valueSet` on an output
channel, the [database client](database.md) records it, and the UI graphs it.

## Devices and Channels

An `iio` node names one device. The channels on that device are detected and
added as `iioChannel` children, so adding a device node and waiting a poll
period is usually the whole setup.

Grouping the channels under the device matches what the kernel does. A channel
is a set of files in a directory shared with every other channel on the device,
alongside settings such as the sample frequency that apply to all of them at
once. Reading them together also means the three axes of an accelerometer belong
to one sample, taken at one moment.

Channels are detected rather than added by hand, because a device exposes
exactly the channels it has and every one of them is something the hardware
measures. A channel can still be added by hand where it is wanted: a driver that
publishes only a converted value with no raw attribute needs a node created for
it, and a person who wants two of a device's eight channels can delete the rest.

## Finding the Device

The devices the kernel has probed appear under `/sys/bus/iio/devices`:

```sh
$ cat /sys/bus/iio/devices/iio:device*/name
ads1015
lsm6dsl
```

`iio_info`, from [libiio](https://github.com/analogdevicesinc/libiio), prints
the channels each one publishes along with their attributes, which is the
quickest way to see what a device will produce.

`Device` accepts the driver's name for the device, the sysfs directory name such
as `iio:device0`, or a full path. Matching by name is preferred, since the
device number depends on probe order and is not stable across boots. The client
publishes the resolved `deviceName` and `devicePath` back to the node either
way, so it is always clear which device is being read.

Many ADCs and sensors need a device tree overlay or an I2C instantiation before
they appear at all. On a Raspberry Pi, an ADS1015 on the default address is
enabled by adding this to `/boot/config.txt`:

```
dtoverlay=ads1015
```

A device whose driver has not probed yet is not an error the client gives up on:
it reports `connected` false with the reason in `Error`, and finds the device on
a later poll once it appears.

## Access to the Device

Reading these attributes requires access to `/sys/bus/iio/devices`. Running
Simple IoT as root grants it; otherwise add its user to a group with access, or
install a udev rule that grants the group you prefer:

```
SUBSYSTEM=="iio", GROUP="iio", MODE="0660"
```

Writing an output channel or a device setting requires write access to the
attribute, which usually needs the same rule.

## Configuration

The device node:

| Field                   | Values                                   | Description                                   |
| ----------------------- | ---------------------------------------- | --------------------------------------------- |
| `Device`                | `ads1015`, `iio:device0`, or a full path | Which IIO device                              |
| `Poll period (ms)`      | milliseconds, default 3000               | How often every enabled channel is read       |
| `Sample frequency (Hz)` | number                                   | Written to `sampling_frequency` when non-zero |
| `Oversampling ratio`    | number                                   | Written to `oversampling_ratio` when non-zero |
| `Disabled`              | boolean                                  | Stop polling without deleting the node        |
| `Debug level (0-9)`     | number                                   | Logs each failed read at level `1` and above  |

A channel node:

| Field        | Values            | Description                                        |
| ------------ | ----------------- | -------------------------------------------------- |
| `Channel`    | `in_voltage0`     | The sysfs attribute prefix, filled in by detection |
| `Scale`      | number, default 1 | Applied to the converted reading                   |
| `Offset`     | number, default 0 | Added after `Scale`                                |
| `Units`      | text              | Defaulted from the channel type, editable          |
| `Min change` | number            | How far the value must move before it is published |
| `Value`      | number            | Writes `valueSet` on an output channel             |
| `Disabled`   | boolean           | Skip this channel and leave the rest reading       |

## Units

The IIO ABI fixes the unit each channel type is reported in. Three of them are
milli units, which the client divides by a thousand so that a rule comparing a
temperature to `25` works the same against an IIO sensor as against a 1-wire
one, and a voltage graphs as `3.3` rather than `3300`:

| Channel type       | ABI unit       | Published as | `Units` |
| ------------------ | -------------- | ------------ | ------- |
| `voltage`          | millivolts     | volts        | `V`     |
| `current`          | milliamps      | amps         | `A`     |
| `temp`             | millidegrees C | degrees C    | `C`     |
| `accel`            | m/s²           | as-is        | `m/s^2` |
| `anglvel`          | rad/s          | as-is        | `rad/s` |
| `magn`             | Gauss          | as-is        | `G`     |
| `pressure`         | kilopascals    | as-is        | `kPa`   |
| `humidityrelative` | percent        | as-is        | `%`     |
| `illuminance`      | lux            | as-is        | `lx`    |
| `proximity`        | unitless       | as-is        | empty   |
| anything else      | driver defined | as-is        | empty   |

`Units` is set from this table when a channel is detected and can be edited
afterward, which is what a channel scaled into engineering units needs.

## Two Layers of Scale

The kernel publishes its own `_scale` and `_offset` attributes, which convert a
raw count into the ABI's physical unit. The client applies these, preferring an
already-converted `_input` attribute where the driver publishes one. This is the
driver's business and needs no configuration.

The node's `Scale` and `Offset` sit above that and convert the physical unit
into the quantity the sensor is actually wired to measure. Keeping them separate
means a person editing a node never has to know the device's full-scale range,
and replacing an ADS1115 with an ADS1015 changes nothing on the node.

### A 4-20 mA Loop

A 4-20 mA transmitter reporting a tank level from 0 to 100 percent, read across
a 100 Ω sense resistor, produces 0.4 V at the bottom of its range and 2.0 V at
the top. Read as a voltage channel, that is:

- `Scale`: 62.5, since a 1.6 V span covers 100 percent
- `Offset`: -25, so that 0.4 V lands on 0
- `Units`: `%`

A reading of 1.2 V then publishes as 50 percent, and a broken loop reading 0 V
publishes as -25, which is visibly out of range rather than a plausible empty
tank.

## Publishing and Min Change

An ADC's low bits dither on every sample, so publishing on any change would
write a point per poll forever. `Min change` sets how far a reading must move
from the last published one before it is sent. This is the setting to reach for
when a channel is filling the database with noise.

Underneath it, the client republishes the value every ten minutes even when
nothing has changed, so a graph and an upstream instance always have a recent
sample.

## Output Channels

A channel detected as `out_*` accepts a `valueSet` point. The client inverts the
conversion chain to a raw count, writes it, reads the channel back, and
publishes `value`. Keeping the two separate means the client's report of the
channel can never be mistaken for a command, and a write that fails leaves
`valueSet` and `value` visibly disagreeing.

Writing `valueSet` on an input channel is reported in the `Error` field rather
than silently ignored.

## Sample Rate

This client polls sysfs, and is built for low-rate readings: a tank level, a
loop current, a board temperature, a battery voltage, read every second or few
seconds and published when it moves. A `Poll period` much below 100 ms is
outside what it is meant to do, and asking for one produces jitter rather than
faster sampling.

`Sample frequency` and `Oversampling ratio` are the settings that actually
improve a reading. Many ADCs convert continuously and a sysfs read returns the
most recent conversion, so the sample frequency decides how stale a reading can
be, and the oversampling ratio trades conversion time for noise. A device that
does not publish one of these settings reports that in the log and is not
counted as an error.

Capturing a waveform is a different problem: it means enabling scan elements,
attaching a trigger, and decoding packed binary records, and it needs a data
model where a point per sample does not overwhelm the store. That is not what
this client does.

## Published Points

| Point         | Type   | Description                                   |
| ------------- | ------ | --------------------------------------------- |
| `value`       | number | The converted reading, on a channel node      |
| `connected`   | bool   | Whether the device was found and is readable  |
| `deviceName`  | text   | The device's `name` attribute                 |
| `devicePath`  | text   | The resolved `iio:deviceN` directory          |
| `channel`     | text   | The attribute prefix, filled in by detection  |
| `channelType` | text   | The measured quantity, filled in by detection |
| `direction`   | text   | `input` or `output`, filled in by detection   |
| `error`       | text   | Why the device or channel could not be read   |
| `errorCount`  | count  | Failed resolutions, reads, and writes         |

## Schema

An `iio` node ready for `siot import`:

```yaml
nodes:
  - iio:
      description: Tank level ADC
      device: ads1015
      pollPeriod: 1000
      sampleFrequency: 128
```

The channels arrive on their own once the node is imported, so what a file adds
is the device and the settings that apply to it. A channel can be described in
the same file where its conversion is worth carrying:

```yaml
nodes:
  - iio:
      description: Tank level ADC
      device: ads1015
      pollPeriod: 1000
      children:
        - iioChannel:
            description: Tank level
            channel: in_voltage0
            channelType: voltage
            direction: input
            scale: 62.5
            offset: -25
            units: "%"
            minChange: 0.5
```
