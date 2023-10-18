# Configuration

## Environment variables

Environment variables are used to control various aspects of the application.
The following are currently defined:

- **General**
  - `SIOT_HTTP_PORT`: http network port the SIOT server attaches to (default
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
    a 0.5s timeout for TLS upgrade, but that is too short for some embedded
    systems that run on low end CPUs connected over cellular modems (we've see
    this process take as long as 4s). See NATS
    [documentation](https://docs.nats.io/nats-server/configuration/securing_nats/tls#tls-timeout)
    for more information.
  - `SIOT_NATS_WS_PORT`: Port to run NATS websocket (default is 9222, set to 0
    to disable)
- **Particle.io**
  - `SIOT_PARTICLE_API_KEY`: key used to fetch data from Particle.io devices
    running [Simple IoT firmware](https://github.com/simpleiot/firmware)

## Configuration import

Nodes defined in a YAML file can be imported into a running SIOT instance using
the CLI, or the Go API. When using the CLI, the import file must be specified on
STDIN. The following imports a new node tree under the root device node. This is
useful for adding new functionality to an instance. If there are any node IDs in
the import they are mapped to new IDs to eliminate any possibility of ID
conflicts if the config is imported into multiple systems with a common upstream
sync, etc.

`siot import < import.yaml`

If you want to import nodes at a specific location (typically a group), then you
can specify the parent node ID. This ID can be obtained by expanding the node
and clicking the copy button. This will put the ID into your system copy buffer.

`siot import --parentID 9d7c1c03-0908-4f8b-86d7-8e79184d441d < import.yaml`

If authentication or a different server is required, this can be specified
through command line arguments or the following environment variables (see
descriptions above):

- `SIOT_NATS_SERVER`
- `SIOT_AUTH_TOKEN`

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

## Configuration export

Nodes can be exported to a YAML file. This is a useful to:

- backup the current configuration
- dump node data for debugging
- transfer a configuration or part of a configuration from one instance to
  another

To export the entire tree:

`siot export > export.yaml`

A subset of the tree can be exported by specifying the node ID:

`siot export -nodeID 9d7c1c03-0908-4f8b-86d7-8e79184d441d > export.yaml`
