# Configuration

## Environment variables

Environment variables are used to control various aspects of the application.
The following are currently defined:

- **General**
  - `SIOT_HTTP_PORT`: HTTP network port the SIOT server attaches to (default
    is 8118)
  - `SIOT_DATA`: directory where any data is stored, including the instance's
    device key in `device.nkey`
  - `SIOT_AUTH_TOKEN`: auth token used for NATS and HTTP device API, default is
    blank (no auth)
  - `SIOT_DEVICE_AUTH`: `optional` (the default) accepts the auth token from
    anywhere; `required` accepts it only from this host, so remote devices need
    a [device credential](sync.md#device-credentials). See the
    [security reference](../ref/security.md#nats).
  - `OS_VERSION_FIELD`: the field in `/etc/os-release` used to extract the OS
    version information. Default is `VERSION`, which is common in most distros.
    The Yoe Distribution populates `VERSION_ID` with the update version, which
    is probably more appropriate for embedded systems built with Yoe. See
    [ref/version](../ref/version.md).
- **NATS configuration**
  - `SIOT_NATS_PORT`: Port to run NATS on (default is 4222 if not set)
  - `SIOT_NATS_HTTP_PORT`: Port to run NATS monitoring interface (default
    is 8222)
  - `SIOT_NATS_SERVER`: defaults to nats://127.0.0.1:4222
  - `SIOT_NATS_TLS_CERT`: points to TLS certificate file. If not set, TLS is not
    used.
  - `SIOT_NATS_TLS_KEY`: points to TLS certificate key
  - `SIOT_NATS_TLS_TIMEOUT`: Configure the TLS upgrade timeout. NATS defaults to
    a 0.5 second timeout for TLS upgrade, but that is too short for some
    embedded systems that run on low end CPUs connected over cellular modems
    (we've see this process take as long as 4 seconds). See NATS
    [documentation](https://docs.nats.io/nats-server/configuration/securing_nats/tls#tls-timeout)
    for more information.
  - `SIOT_NATS_WS_PORT`: Port to run NATS WebSocket (default is 9222, set to 0
    to disable)
  - `SIOT_NATS_MQTT_PORT`: Port to serve MQTT on (disabled by default; 1883 is
    the conventional port). See the [MQTT page](mqtt.md).
- **Provisioning**
  - `SIOT_PROVISIONING_DIR`: directory of YAML files applied at start-up and
    whenever they change. If it is not set, `<SIOT_DATA>/provisioning` is used
    when that directory exists, so an image can ship the directory and say
    nothing else.
  - `SIOT_PROVISIONING_INTERVAL`: how often to look for changes the directory
    watch and the tree subscription might have missed, written as a Go duration
    such as `60s`. The default is one minute.
- **Particle.io**
  - `SIOT_PARTICLE_API_KEY`: key used to fetch data from Particle.io devices
    running [Simple IoT firmware](https://github.com/simpleiot/firmware)

## The configuration file format

One format describes a tree of nodes, and `siot export`, `siot import`, and
provisioning all use it. The node type is the key, and each point type is a key
of its own:

```yaml
apiVersion: 1
nodes:
  - group:
      description: Sensors
      children:
        - modbus:
            description: Modbus sensors
            port: /dev/ttyS1
            baud: 9600
            debug: 0
```

How a value is written decides what it becomes:

| YAML value               | Point                                            |
| ------------------------ | ------------------------------------------------ |
| string (`hello`, `"10"`) | text                                             |
| integer (`10`)           | integer value                                    |
| float (`1.5`)            | float value                                      |
| bool (`true`)            | value 1 or 0                                     |
| null                     | a point with no value                            |
| mapping                  | one point per entry, the key becomes a point key |
| sequence                 | one point per element, keyed `"0"`, `"1"`, …     |

Quoting is what tells a text value from a numeric one, which matters when a
value looks like a number: `port: 502` is numeric and `port: "502"` is text. If
a client expects text and the file gives it a number, the client reads an empty
value, so quote anything that is really text.

A mapping under a point type is a set of keyed points, and a sequence is an
array:

```yaml
- metrics:
    metricSysCPUFreq:
      cpu0: 1400
      cpu1: 1600
    tag: [alpha, beta] # keys "0" and "1"
```

Three keys inside a node are reserved: `parent`, `children`, and `edgePoints`.
Every other key is a point type, `id` included -- Modbus and OneWire nodes
configure a point named `id`, and it is written like any other point. A node's
own ID never appears in a file.

Edge points, such as a user's role, are spelled the same way under their own
key:

```yaml
- user:
    firstName: Admin
    email: admin@example.com
    edgePoints:
      role: admin
```

Points that a file does not need to carry are left out of an export. The
`nodeType` edge point is one of them, since the node type is the key each node
is written under, and the system fills it in when a file is applied.

### How nodes are found

A file describes what the tree should look like rather than naming the nodes it
means by ID, so applying one twice does what applying it once did. A node in a
file matches an existing node when the parent and the description agree:

- **No match:** the node is created.
- **A match of the same type:** only the points whose values differ are sent.
- **A match of a different type:** an error, since a file that says `modbus`
  where the tree holds a `group` is either a mistake or a rename.
- **More than one match:** an error, since nothing says which node was meant.

A user node has no description, so an email address identifies it, and a name if
there is no email. An entry with no description at all matches the single node
of its type, which is how a `metrics` or `serial` node is addressed.

**A description is how a file finds a node.** Renaming one in the UI detaches it
from the file that describes it, and the next time that file is applied it
creates a second node beside the renamed one. The same is true of renaming a
node in a file. Renaming deliberately is a two step change: delete the old
description in the same file that introduces the new one. Give nodes
descriptions that are meant to last.

### Where nodes attach

A top level entry with no `parent` is applied under this instance's device node.
A `parent` names a node anywhere in the tree by description, which is how a file
adds to a subtree it did not create:

```yaml
nodes:
  - group:
      description: Tank farm
  - variable:
      parent: Tank farm
      description: Tank level
```

Entries apply in the order they are written, so a `parent` naming a node the
same file creates has to come after the entry that creates it.

### Referring to another node

A point of type `nodeID` names the node it refers to by description, and is
resolved the same way `parent` is:

```yaml
nodes:
  - variable:
      description: Tank level
  - rule:
      description: Tank low
      children:
        - condition:
            description: Level below 10
            nodeID: Tank level
            operator: "<"
            value: 10
```

References resolve after the whole file has been read, so one may point at a
node the file creates further down, or at a node another file created.

### Removing nodes

Applying a file adds and updates; it never removes something for going
unmentioned. A `delete` list removes nodes, matched the way `nodes` entries are:

```yaml
delete:
  - modbus:
      parent: Tank farm
      description: Old sensors
```

Deleting what is already gone does nothing, so a file with a `delete` list is as
safe to apply repeatedly as any other.

## Configuration export

Nodes can be exported to a YAML file. This is useful to:

- Back up the current configuration
- Transfer a configuration, or part of one, from one instance to another
- Build a configuration in the UI and then ship it as a provisioning file

To look at an instance rather than reproduce it, use `siot dump` instead, which
is described below.

To export the entire tree:

`siot export > backup.yaml`

A subset of the tree can be exported by specifying the node ID:

`siot export -nodeID 9d7c1c03-0908-4f8b-86d7-8e79184d441d > export.yaml`

An export describes configuration and nothing else, which is what makes it
usable as a provisioning file:

- The root node is left out. It is this instance rather than configuration, and
  a file describing it would match nothing anywhere else. Exporting the tree
  exports what is under the root.
- Node IDs are left out, since a file finds its nodes by description. A `nodeID`
  point is written as the description of the node it points at.
- Points that carry no value are left out, as is the origin recording which
  client last wrote each point.

Two nodes that share a parent and a description cannot be told apart by a file,
so `siot export` reports that rather than writing a file that would do the wrong
thing when applied. Give those nodes distinct descriptions, which is worth doing
anyway.

## Instance dump

`siot dump` describes an instance as it actually is. Export answers "what would
recreate this configuration"; dump answers "why is this instance behaving the
way it is", so it reports the identifiers and structure export leaves out:

`siot dump`

- The instance root node ID, which is the identity this instance replicates
  under
- The tree with every node ID and type, including deleted nodes
- Every parent of each node, so a node that appears in more than one place says
  so
- An `anomalies` section listing any node other than the root that carries the
  virtual `root` parent, which would give the instance a second root

Two flags add detail:

- `siot dump -points` includes every point with the origin that wrote it and the
  time it was written, which is what to compare when two instances disagree
  about a value
- `siot dump -streams` lists the boundary-origin replication streams and their
  message counts, which shows at a glance which instances this one replicates
  with

`siot dump -all` turns on both, and `siot dump -nodeID <id>` limits the tree to
one subtree.

Comparing the same dump from two instances is the quickest way to tell a
replication problem from a configuration one. Instances that disagree about
their root IDs, or that are missing a stream for each other, have a replication
problem; instances that agree on structure but differ on a point's origin or
time have a configuration one.

## Configuration import

`siot import` applies a file to a running instance, reading it from `STDIN`:

`siot import < config.yaml`

Nodes are matched by description, as described above, so importing a file
creates what is missing, updates what has drifted, and does nothing when the
tree already agrees. Importing the same file twice does what importing it once
did.

`siot import -dryRun < config.yaml` prints what the file would do without
applying any of it.

If authentication or a different server is required, this can be specified
through command line arguments or the following environment variables (see
descriptions above):

- `SIOT_NATS_SERVER`
- `SIOT_AUTH_TOKEN`

`siot import --help` for more details.

Example YAML file:

```yaml
nodes:
  - group:
      description: group 1
      children:
        - variable:
            description: var 1
            value: 10
```

## Configuration provisioning

An instance can be configured from files rather than by hand. Provisioning
applies the same files `siot import` does, and applies them at start-up and
whenever they change, so a unit built from an image comes up configured with no
import step and no operator involvement.

There are two places files come from:

- **A directory on disk**, given by `-provisioningDir` or
  `SIOT_PROVISIONING_DIR`, defaulting to `<SIOT_DATA>/provisioning` when that
  directory exists. Files are applied in lexical order, so the familiar `10-`,
  `20-` prefixes express which file goes first.
- **Files uploaded through the UI**, which are `file` nodes under the
  `provisioning` node. This is how a unit whose filesystem you cannot reach gets
  configured.

Files on disk are applied first and uploads layer on top, so an uploaded file
can attach to a group a shipped file created. Uploads are applied oldest first,
by when the file was added rather than when its contents were last replaced, so
correcting a file does not change its place in the order.

A file node exists from the moment it is added and its contents arrive when you
upload them, so provisioning waits for the upload and leaves an empty file node
alone.

A file is applied when its contents change, which is what leaves a value edited
in the UI alone until the file describing it changes. A file that fails to parse
or apply records its error and leaves the other files alone.

### Checking files

`siot provision -dir ./provisioning` prints what the files in a directory would
do to a running instance without applying any of it.

`siot provision -dir ./provisioning -check` only parses them, which needs no
running instance and is what a build can use to fail on a bad file.

### Seeing what happened

A `provisioning` node under the root records what was applied. Each file on disk
gets a `provisioningFile` child carrying its name, the checksum of what was
applied, and the last error if it failed. An uploaded file records the same
thing on the file node itself, so a file and its status are one node in the UI.

Removing a file from the directory removes its status. The nodes it created stay
where they are: provisioning describes what should exist, and does not own what
it made. Use a `delete` list to remove nodes.
