# Configuration

## Environment variables

Environment variables are used to control various aspects of the application.
The following are currently defined:

- **General**
  - `SIOT_HTTP_PORT`: HTTP network port the SIOT server attaches to (default
    is 8118)
  - `SIOT_DATA`: directory where any data is stored
  - `SIOT_AUTH_TOKEN`: auth token used for NATS and HTTP device API, default is
    blank (no auth)
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
- **Particle.io**
  - `SIOT_PARTICLE_API_KEY`: key used to fetch data from Particle.io devices
    running [Simple IoT firmware](https://github.com/simpleiot/firmware)

## Configuration export

Nodes can be exported to a YAML file. This is a useful to:

- Back up the current configuration
- Dump node data for debugging
- Transfer a configuration or part of a configuration from one instance to
  another

To export the entire tree:

`siot export > backup.yaml`

A subset of the tree can be exported by specifying the node ID:

`siot export -nodeID 9d7c1c03-0908-4f8b-86d7-8e79184d441d > export.yaml`

## Configuration import

Nodes defined in a YAML file can be imported into a running SIOT instance using
the CLI, or the Go API. When using the CLI, the import file must be specified on
`STDIN`. If there are any node IDs in the import they are mapped to new IDs to
eliminate any possibility of ID conflicts if the config is imported into
multiple systems with a common upstream sync, etc.

If nodes reference each other (for instance a rule condition and a Modbus node),
then friendly IDs can be used to make it easy to edit and reference. These
friendly IDs will be replaced by a common UUID during import.

To import nodes at a specific location (typically a group), then you can specify
the parent node ID. This ID can be obtained by expanding the node and clicking
the copy button. This will put the ID into your system copy buffer.

`siot import -parentID 9d7c1c03-0908-4f8b-86d7-8e79184d441d < import.yaml`

If you want to wipe out any existing state and restore a SIOT to a known state,
you can run an import with the `-parentID` set to `root`. It is highly
recommended you restart SIOT after this is done to minimize the chance of any
code still running that caches the root ID which has now changed.

`siot import -parentID root < backup.yaml`

Again, by default, the import command will create new IDs to minimize the chance
of any ID conflicts. If you want to preserve the IDs in the YAML file, you can
specify the `-preserveIDs` option - **WARNING**, use this option with caution.
Importing a backup to `root` with `-preserveIDs` is a handy way to restore a
system to a known previous state. However, new nodes that don't exist in the
backup will not be deleted - the import only adds nodes/points.

If authentication or a different server is required, this can be specified
through command line arguments or the following environment variables (see
descriptions above):

- `SIOT_NATS_SERVER`
- `SIOT_AUTH_TOKEN`

**It is easy to make a mess with the import command, so think through what you
are doing first. SIOT does not prevent you from making a mess!**

`siot import --help` for more details.

Example YAML file:

```yaml
nodes:
  - type: group
    points:
      - type: description
        text: "group 1"
    children:
      - type: variable
        points:
          - type: description
            text: var 1
          - type: value
            value: 10
```
