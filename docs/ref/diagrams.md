# Diagrams

<!-- toc -->

Diagrams in the SIOT documentation are plain SVG files written by hand. Each
file is its own source, so there is nothing to export: edit the SVG, view it in
a browser, and commit it. The shared style below keeps every diagram consistent
and lets each one follow the reader's light or dark theme.

## Style guide

### Starting a new diagram

Copy an existing diagram that is close to what you need and edit it. Every file
starts with the same `<style>` block, `<defs>` with the arrow markers, and a
background `<rect class="bg">`. Keep that header unchanged so the diagrams stay
in step; if the style needs to change, change it in every file.

- Put the file in the `images/` directory next to the page that uses it (ADRs
  use `assets/`).
- Set `viewBox`, `width`, and `height` to the same size, about 900 × 500 or
  smaller.
- Describe the diagram in the `aria-label` on the `<svg>` element, and give the
  Markdown image the same kind of alt text.
- Leave the title out of the diagram. The heading or sentence around it says
  what it shows.

### Shapes

| Element       | SVG                                       | Use                                                        |
| ------------- | ----------------------------------------- | ---------------------------------------------------------- |
| Box           | `<rect class="shape blue" rx="6">`        | a thing: an instance, a client, a node, a device           |
| Container     | `<rect class="shape gray" rx="10">`       | where things run or belong: an instance, a site, the cloud |
| Pill          | `<rect class="shape purple">`, `rx = h/2` | a NATS bus or a stream                                     |
| Circle        | `<circle class="shape green">`            | a node in an abstract graph (`N1`, `a`)                    |
| Diamond       | `<polygon class="shape yellow">`          | an edge that carries points                                |
| Stick figure  | `<g class="actor">`                       | a person                                                   |
| Dashed border | add `dashed`                              | a replica, a mirror, or something that does not run        |

Container titles start with an emoji so the reader can tell places apart at a
glance: ☁️ cloud, 📟 edge device, 🏭 site, 📦 SIOT instance, 🧩 clients. Use an
emoji in a box label only when it names a kind of thing (👤 user, 📜 rule).

### Colors

Each color has a meaning. Pick the color by what the shape is, never for
decoration.

| Class    | Light fill | Meaning                                      |
| -------- | ---------- | -------------------------------------------- |
| `blue`   | `#dae8fc`  | SIOT instances, clients, and other software  |
| `green`  | `#d5e8d4`  | nodes in the tree                            |
| `yellow` | `#fff2cc`  | points and other data                        |
| `purple` | `#e1d5e7`  | NATS, streams, and external services         |
| `orange` | `#ffe6cc`  | hardware: sensors, MCUs, motors              |
| `red`    | `#f8cecc`  | a warning, or something that does not happen |
| `gray`   | `#f5f5f5`  | containers                                   |

The `@media (prefers-color-scheme: dark)` rule in the style block swaps each
fill for a dark tint of the same color, so the meaning carries over to the dark
theme.

### Lines

All lines use `class="edge"` (1.5 px):

- **Flow**: an arrowhead at the end, `marker-end="url(#arrow)"`.
- **Two-way**: add `marker-start="url(#arrow)"`, for sync or request/response.
- **Tree**: no arrowheads, drawn as elbows from parent to child, with the parent
  above or to the left.
- **Dashed** (`edge dashed`): a logical link, such as a subscription or the same
  node in two places.
- **Colored** (`edge c-red` with `url(#arrow-red)`): only when one diagram shows
  more than one kind of flow; say what each color means in a label.

### Text

| Class         | Size | Use                                          |
| ------------- | ---- | -------------------------------------------- |
| `title`       | 14px | container titles                             |
| _(default)_   | 12px | box labels                                   |
| `small`       | 11px | line labels and second lines in boxes        |
| `small muted` | 11px | notes and explanations                       |
| `mono`        | 11px | stream names, NATS subjects, and point types |

Line labels sit on a `<rect class="label-bg">` so they stay readable where they
cross a line. Use sentence case and keep labels to a few words; put longer
explanations in the page text.

### Checking a diagram

Open the SVG in a browser and switch the system between light and dark mode. To
check from the command line, render both modes with Chromium:

```sh
chromium --headless --force-device-scale-factor=1 --window-size=900,500 \
  --screenshot=light.png file://$PWD/docs/ref/images/arch-app.svg
chromium --headless --force-device-scale-factor=1 --window-size=900,500 \
  --force-dark-mode --blink-settings=preferredColorScheme=0 \
  --screenshot=dark.png file://$PWD/docs/ref/images/arch-app.svg
```

## Gallery

Every diagram in the documentation, for reference and review. When you add a
diagram, add it here too.

### Application architecture

![application architecture](images/arch-app.svg)

Used in [Architecture](architecture.md), [Application](architecture-app.md), and
[Client](client.md).

### Distributed system

![cloud instances synchronized with edge devices, browsers, and phones](images/distributed.svg)

Used in [System](architecture-system.md).

### Data flow

![clients write points to NATS; the store persists them](images/data-flow.svg)

Used in [System](architecture-system.md).

### Constant and varying parts of a system

![common core with varying UI and device software](images/constant-vs-varying-system-parts.svg)

Used in [System](architecture-system.md).

### Cloud and device node trees

![device instances hold subtrees of the cloud node tree](images/cloud-device-node-tree.svg)

Used in [System](architecture-system.md) and
[ADR-7](../adr/7-jetstream-store.md).

### Example node tree

![example node tree with customers, users, devices, and rules](images/nodes2.svg)

Used in [System](architecture-system.md).

### Directed acyclic graph

![a directed acyclic graph of nodes](images/dag.svg)

Used in [Data](data.md) and [ADR-7](../adr/7-jetstream-store.md).

### Primary and mirror edges

![primary and mirror edges](images/primary-mirror-edges.svg)

Used in [Data](data.md) and [Client](client.md).

### Sync streams

![device and hub streams](images/sync-streams.svg)

Used in [Synchronization](sync.md).

### MCU nodes and points

![nodes and points for an MCU device](images/mcu-nodes.svg)

Used in [Serial Devices](serial.md).

### Use cases

![SIOT in the cloud connected to edge sites and remote users](../user/images/use.svg)

![edit configuration from the cloud or on site](../user/images/edit-anywhere.svg)

![SIOT integrated with databases, scripts, and hardware](../user/images/integration.svg)

![edge gateways sync through local servers to the cloud](../user/images/multiple-upstream.svg)

Used in [Use Cases](../user/use-cases.md), [Integration](integration.md), and
[Synchronization](../user/sync.md).

### MCU over USB serial

![SIOT on a Raspberry Pi talking to an MCU over USB serial](../user/images/mcu.svg)

Used in [MCU Devices](../user/mcu.md) and [Serial Devices](serial.md).

### ADR-7: JetStream store

![node points in a stream](../adr/assets/node-stream.svg)

![hub and leaf streams](../adr/assets/hub-leaf-streams.svg)

![edges stored in streams](../adr/assets/edge-streams.svg)

![up subjects](../adr/assets/up-subjects.svg)

Used in [ADR-7](../adr/7-jetstream-store.md).

### Node, point, and edge

![a parent and child node joined by an edge, with the fields of one point](images/node-point-edge.svg)

Used in [Data](data.md).

### Store boundaries

![nodes grouped into hub and device boundaries and their streams](images/store-boundaries.svg)

Used in [Store](store.md).

### Merging subject tips

![the newest tip of two streams becomes the current point](images/store-tip-merge.svg)

Used in [Store](store.md).

### Life of a sync connection

![sequence of a device connecting, being adopted, syncing, and catching up](images/sync-connection.svg)

Used in [Synchronization](sync.md).

### Device enrollment

![a device enrolls with a token, waits for approval, then syncs with its own key](../user/images/sync-enrollment.svg)

Used in [Synchronization](../user/sync.md).

### Who connects

![each kind of connection is checked by the authorizer and granted a scope](images/auth-principals.svg)

Used in [Authentication and authorization](auth.md) and [Security](security.md).

### Browser connections

![the browser uses HTTP for sign-in and node operations and NATS over WebSocket for data](images/browser-paths.svg)

Used in [Frontend](frontend.md).

### High availability approaches

![active/passive, active/active, and peers replicating through sync](images/high-availability-approaches.svg)

Used in [High Availability](high-availability.md).

### Notification delivery

![a notification becomes messages that each messaging service deduplicates, rate limits, and delivers](images/notification-delivery.svg)

Used in [Notifications](notifications.md).

### Bandwidth budget

![monthly bytes a syncing device sends, per source](images/bandwidth-budget.svg)

Used in [Bandwidth](bandwidth.md).

### PLC integration paths

![the ways PLC data reaches Simple IoT](../user/images/plc-paths.svg)

Used in [PLCs](../user/plc.md).

### MQTT topic schema

![each named topic level becomes a node with a tag](../user/images/mqtt-topic-schema.svg)

Used in [MQTT](../user/mqtt.md).

### IIO scaling

![a 4-20 mA reading scaled by the driver and then by the node](../user/images/iio-scale.svg)

Used in [IIO](../user/iio.md).

### Configuration provisioning

![provisioning files applied to the node tree with their status recorded](../user/images/provisioning.svg)

Used in [Configuration](../user/configuration.md).

### ADR-9: file transfer and updates

![file contents uploaded in the cloud, fetched by the device, and installed](../adr/assets/file-fetch-ota.svg)

Used in [ADR-9](../adr/9-large-files-over-nats.md).
