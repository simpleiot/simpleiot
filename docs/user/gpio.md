# GPIO Client

The GPIO client reads or drives a single line on a Linux GPIO character device.
A door switch, a float switch, a pump enable, a status LED, and an alarm relay
are all one line each, and each one is a `gpio` node.

An input publishes a `value` point whenever the line changes. An output takes a
`valueSet` point, drives the line, reads it back, and publishes `value`. Both
work with everything downstream of a point: a rule condition can watch `value`,
a rule action can write `valueSet`, the [database client](database.md) records
both, and the UI graphs them.

## One Node Per Line

There is no chip node with lines under it. Each line is an independent request
on the chip with its own file descriptor, its own edge stream, and its own
settings, so each line is a node of its own. Two things follow from that:

- A `gpio` node lives next to the thing it controls. The pump enable goes in the
  pump group and the door switch goes in the door group, rather than under a
  heading that mirrors the board.
- Editing one line disturbs only that line. Adding or reconfiguring a line
  leaves every other line on the chip requested and holding its state.

Lines are added deliberately rather than detected. A chip exposes every line the
SoC has, which is 54 on a Raspberry Pi and over a hundred on some parts, and
nearly all of them are either unrelated to the application or already claimed by
a driver. Add a node for each line the application actually uses.

## Finding the Chip and the Line

`gpiodetect` and `gpioinfo`, from
[libgpiod](https://git.kernel.org/pub/scm/libs/libgpiod/libgpiod.git/), list
what a board offers:

```sh
$ gpiodetect
gpiochip0 [pinctrl-bcm2835] (54 lines)

$ gpioinfo gpiochip0
gpiochip0 - 54 lines:
        line   0:  "ID_SDA"   unused   input  active-high
        ...
        line  17:  "GPIO17"   unused   input  active-high
        line  18:  "GPIO18"   "my-driver" input active-high [used]
```

`Chip` accepts a chip name such as `gpiochip0`, a full device path such as
`/dev/gpiochip0`, or the chip's label, which is the name in brackets in the
`gpiodetect` output. A label is worth using where an expander does not always
land on the same chip number between boots.

`Line` accepts either the line offset or the kernel's name for the line, so `17`
and `GPIO17` select the same line above. Naming the line is more durable, since
offsets move between kernel versions and board revisions. The client publishes
the resolved `lineOffset` and `lineName` back to the node either way, so it is
always clear which line is held.

On a Raspberry Pi, the 40-pin header is on `gpiochip0` on most kernels. The Pi 5
moved it, which is a good illustration of why the chip is configurable rather
than assumed.

## Access to the Device

Requesting a line requires access to `/dev/gpiochipN`. Running Simple IoT as
root grants it; otherwise add its user to the group that owns the device, which
is usually `gpio`, or install a udev rule that grants the group you prefer:

```
SUBSYSTEM=="gpio", KERNEL=="gpiochip*", GROUP="gpio", MODE="0660"
```

A line already claimed by a driver cannot be requested. The `Error` field on the
node names the driver holding it, which is usually enough to identify the device
tree overlay or module to change.

## Configuration

| Field               | Values                                        | Description                                                  |
| ------------------- | --------------------------------------------- | ------------------------------------------------------------ |
| `Chip`              | `gpiochip0`, a label, a path, or `sim`        | Which GPIO chip the line is on                               |
| `Line`              | offset or kernel line name                    | Which line on that chip                                      |
| `Direction`         | `input` (default), `output`                   | Whether the client reads the line or drives it               |
| `Bias`              | as is (default), pull up, pull down, disabled | Internal bias, which applies mainly to inputs                |
| `Drive`             | push-pull (default), open drain, open source  | Output drive mode                                            |
| `Active low`        | boolean                                       | Invert the line: a low line reads and drives as active       |
| `Debounce (ms)`     | milliseconds                                  | Kernel debounce for edge events, inputs only                 |
| `Poll period (ms)`  | milliseconds                                  | Non-zero switches an input from edge events to polling       |
| `Initial value`     | boolean                                       | The state an output is driven to when the line is requested  |
| `Value`             | boolean                                       | Drives an output through `valueSet` and shows the line state |
| `Disabled`          | boolean                                       | Release the line without deleting the node                   |
| `Debug level (0-9)` | number                                        | Logs each edge event at level `1` and above                  |

Bias, debounce, and per-line configuration require Linux 5.5 or later; debounce
in particular requires 5.10. On an older kernel these settings are ignored or
the request fails, depending on the setting and the driver.

### Edge Events and Polling

An input is requested with both edges and an event handler, so a change reaches
the point stream in about a millisecond with no poll timer running. This is the
default and is what most lines should use.

Some lines cannot deliver edge events: expanders behind an I2C bridge, chips
still on version 1 of the kernel interface, and kernels without interrupt
support for the pin. Setting `Poll period` to a non-zero value switches the line
to a periodic read instead.

Either way, the client reads and publishes the line as soon as it is requested,
because edge events only report changes. It also republishes the value every ten
minutes even when nothing has changed, so a graph or an upstream instance always
has a recent sample.

### Outputs

Writing `valueSet` drives the line; the client then reads the line back and
publishes `value`. Keeping the two separate means the client's report of the
line state can never be mistaken for a command, and a write that fails leaves
`valueSet` and `value` visibly disagreeing.

`Initial value` is what the line is driven to when it is requested, which
includes every restart of Simple IoT and every configuration change. Set it to
the safe state for whatever the line controls.

Writing `valueSet` on an input is reported in the `Error` field rather than
silently ignored.

### Recovering From a Failed Request

When a request fails, the node reports `connected` false, the reason in `Error`,
and a rising `errorCount`, and the client retries with a growing delay. A line
held by a driver that has not finished loading recovers on its own, and a line
named incorrectly recovers as soon as the name is corrected, with no need to
restart anything.

`Disabled` releases the line, which is the way to hand a line back to the kernel
without deleting the node. What an output line does after it is released is up
to the driver: some controllers return the line to an input, and others leave it
configured and driven where it was. A Raspberry Pi 5 does the latter, so a
released output holds its last level until something else claims the line. Where
the state of a line matters when Simple IoT is not holding it, set it
deliberately before releasing it rather than relying on the release.

## Trying It Without Hardware

Setting `Chip` to `sim` gives the node a simulated line instead of a kernel one.
Simulated lines are keyed by offset, and writing a simulated output delivers an
edge to every simulated input at the same offset, which behaves like a wire
between them. Two nodes at offset `1`, one an output and one an input, are
enough to develop and test a rule before any hardware exists.

The simulated chip names its line at offset 7 `sim7`, so a simulated line can be
selected by name as well as by offset.

## Published Points

| Point        | Type    | Description                                             |
| ------------ | ------- | ------------------------------------------------------- |
| `value`      | boolean | The line state as read back                             |
| `connected`  | boolean | Whether the line is currently requested                 |
| `lineOffset` | number  | Resolved offset, useful when `line` was given as a name |
| `lineName`   | text    | The kernel's name for the resolved line                 |
| `error`      | text    | Why the last request or access failed                   |
| `errorCount` | count   | Failed requests, reads, and writes                      |

## Schema

Below is an export of two `gpio` nodes, a relay output and a switch input:

```yaml
nodes:
  - gpio:
      description: Pump enable
      chip: gpiochip0
      line: "17"
      direction: output
      initialValue: 0
  - gpio:
      description: Float switch
      chip: gpiochip0
      line: FLOAT_SW
      direction: input
      bias: pullUp
      debounce: 20
```
