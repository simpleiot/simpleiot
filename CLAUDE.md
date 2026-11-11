# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with
code in this repository.

## Project Overview

Simple IoT is a Go-based IoT platform with a Svelte frontend that enables
distributed sensor data collection, telemetry, configuration, and device
management. The system runs the same application in both cloud and edge
instances, automatically synchronizing data between them using NATS messaging.

## Build System and Common Commands

### Setup and Dependencies

```bash
# Source environment setup (required for all operations)
source envsetup.sh

# Initial setup - installs frontend deps
siot_setup

# Install frontend dependencies only
siot_install_frontend_deps
```

### Building

```bash
# Build everything (frontend + backend)
siot_build

# Build frontend only (the Svelte UI)
siot_build_frontend

# Build the Svelte UI only
siot_build_ui

# Build backend only (Go binary)
siot_build_backend

# Cross-compile for ARM
siot_build_arm
siot_build_arm64
```

### Development and Testing

```bash
# Start development mode with hot reloading (both frontend and backend)
siot_watch

# Run the application locally
siot_run [arguments]

# Run complete test suite (frontend + backend + linting)
siot_test

# Test individual components
siot_test_frontend
siot_test_ui          # Svelte UI: prettier, eslint, svelte-check, vitest
siot_test_ui_browser # opt-in Playwright smoke test against a scratch siot
go test -race ./...
```

### Linting and Code Quality

```bash
# Backend linting (uses golangci-lint with revive and goimports)
golangci-lint run

# Svelte UI checks (prettier, eslint, svelte-check, vitest)
siot_test_ui
```

## Architecture

### Core Concepts

- **Nodes**: Core data structures containing arrays of Points
- **Points**: Individual data values with timestamps and metadata
- **Graph Structure**: Data organized as a DAG (directed acyclic graph)
- **IDs**: Opaque strings. Generate new ones with `data.NewID()` (Crockford
  base32), not `uuid`; an ID chosen by hand goes through `data.ValidID()`
- **Clients**: Independent components that implement specific functionality
- **NATS Messaging**: All data flows through embedded NATS message bus

### Key Directories

- `cmd/siot/` - Main application entry point
- `server/` - Server core functionality and HTTP API
- `client/` - Client implementations (most functionality lives here)
- `api/` - HTTP API handlers and routing
- `data/` - Core data structures (Node, Point, etc.)
- `store/` - JetStream storage layer (boundary-origin streams, see ADR-7)
- `ui/` - Svelte web UI, served at `/`
- `js/simpleiot-js/` - the browser client the Svelte UI bundles
- `modbus/` - Modbus protocol implementation
- `network/` - Network management utilities

### Client Architecture

Most functionality is implemented as clients that:

- Subscribe to relevant node changes via NATS
- Process data and implement business logic
- Publish point updates back to the system
- Are managed by the ClientManager system

Common client types: SerialDev, CanBus, Rule, Db, SignalGenerator, Sync,
Metrics, Modbus, OneWire, GPIO, IIO, Mqtt, Shelly, Particle, User, MsgService,
etc.

A node can sit in several places in the tree, and a client runs on the node's
primary edge and on edges with no role, never on a mirror. Adding a client means
classifying its node type in `primaryTypes` or `treeScopedTypes` in
`data/edge_role.go`; a test fails if it is in neither. See
[Primary and mirror edges](docs/ref/data.md#primary-and-mirror-edges).

### Authentication

Every connection to the embedded NATS server (NATS, WebSocket, MQTT) goes
through the authorizer in `server/auth.go`. The shared token grants full access;
an NKey whose public key is in a `deviceCred` node is scoped to that device's
subjects (`devicePermissions`), a user's sign-in JWT is scoped to the user's
groups (`userPermissions`, enforced per request by `handleUserRequest` in
`store/store.go`), and an enrollment token may only publish `enroll.request`.
Each instance keeps its own key in `SIOT_DATA/device.nkey`
(`server/device-key.go`), never as a point, since points replicate upstream; the
seed stays in the server process, which signs for the sync client on
`auth.deviceSign`. The HTTP node routes (`api/nodes.go`) resolve every request
to a principal and apply the same scope. Clients run on the server's full-access
connection, so a value from a point that names a target goes through
`checkWriteTarget` in `client/scope.go`, a period from a point through
`pointDuration`, and a path from a point through `client/paths.go`; secret point
types are listed once in `data/secret.go`. `server/auth_spike_test.go` is the
canary for nats-server upgrades. See [security](docs/ref/security.md) and
[device credentials](docs/user/sync.md#device-credentials).

### Frontend Architecture

The Svelte UI (`ui/`) is a SvelteKit app built with `adapter-static`, embedded
by `ui/embed.go`, and served at `/` by `api/ui.go`:

- **Node types register themselves**: each type is a directory in
  `ui/src/lib/nodes/<type>/` with an `index.ts` descriptor (where it may be
  added, children, owning parent, sort order) and an `Editor.svelte`. Adding a
  node type is adding a directory; a test fails if a type in `schema.ts` has no
  editor, or if a file outside `src/lib/nodes` steers behavior by a type name.
- **Editors are props-only** and built from the input kit in
  `ui/src/lib/inputs/`; only the page and the store
  (`ui/src/lib/siot/tree.svelte.ts`) talk to the server. The kit's code editor
  (`CodeInput`) loads CodeMirror on demand from `ui/src/lib/inputs/code/`, so
  no page carries it until a file is opened.
- **Dashboard widgets register themselves** the same way, one directory per
  widget in `ui/src/lib/widgets/<name>/` with a descriptor and a component that
  takes resolved values as props. The widget an `auto` component shows for a
  node type is the `widget` field on that type's descriptor, so no file outside
  `src/lib/nodes` names a type. The dashboard pages live in
  `ui/src/lib/nodes/dashboard/` for the same reason. A widget that draws a
  document names the binding holding it as `source` and reads further bindings
  and ports from it through `bindingsIn` and `portsIn`; the `svg` widget
  sanitizes the document with DOMPurify first, and every uploaded markup must.
  Code copied from the nautilus HMI kit sits in `ui/src/lib/vendor/nautilus/`
  with its license.
- **Constants**: `ui/src/lib/siot/schema.ts` mirrors `data/schema.go`, and
  `data/schema_ts_test.go` checks it, so renaming a point or node type means
  changing both.
- **Checks**: strict TypeScript, `svelte-check` fails the build, vitest unit
  tests, prettier and eslint.

## Design Priorities

The project is pre-1.0, so there is no backwards compatibility. Make the clean
change and do not add compatibility shims, fallbacks, transition modes, or flags
whose only purpose is to keep an older client, wire format, schema, or
deployment working. Changing a subject format, renaming a point type, or
reshaping stored data is fine when it makes the system simpler or more correct.
When a change is incompatible with an existing deployment, say so in the
`CHANGELOG.md` entry and in any affected documentation so users upgrading know
what to expect. Plans list the incompatibilities to note in the changelog rather
than a compatibility section.

## Development Workflow

1. **Setup**: `source envsetup.sh && siot_setup`
2. **Development**: `siot_watch` (starts hot reloading for both frontend and
   backend)
3. **Testing**: `siot_test` before submitting changes
4. **Code Quality**: All code must pass `golangci-lint run` and `siot_test_ui`

## Plans

Implementation plans are stored in the `plans/` directory. See `plans/plans.md`
for an index of all plans and their status.

When working through a plan, commit after each phase completes. Update the
changelog (`CHANGELOG.md`), `CLAUDE.md`, and any relevant documentation as part
of each phase.

## Diagrams

Documentation diagrams are hand-written SVG files, not draw.io. Copy an existing
SVG to keep the shared style header, follow the style guide in
[docs/ref/diagrams.md](docs/ref/diagrams.md), and add each new diagram to the
gallery on that page.

## Changelog

The changelog uses [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)
format — add entries under the `## [Unreleased]` section.

Keep entries concise. Start each one with a **bold summary sentence** naming
what changed, then at most two or three sentences of detail:

```markdown
- **JetStream streams are compressed with S2, on by default.** A store of
  100,000 scraped points went from 33.4 MB to 11.7 MB in testing.
  `--storeCompression` (or `SIOT_STORE_COMPRESSION`) accepts `s2` or `none`. See
  the [store reference](docs/ref/store.md).
```

A reader is scanning to find what affects them, so the summary has to carry the
entry on its own. Keep the detail to what someone upgrading needs — a changed
default, a new setting, a link to the documentation. Reasoning, measurements,
and design background belong in the docs and the commit message, not here. Call
out anything that changes behavior for existing users explicitly.

## Branching

Make changes on the branch that is already checked out. Do not create a feature
branch, switch branches, or start a worktree unless I ask for one.

## Committing

Do not commit the UI build assets as part of a code change. The Svelte UI build
in `ui/build` is rebuilt and committed by the user before a release. Leave it
out of staging even when `ui/` changes; if `siot_build_ui` was run, do not stage
the result.

Stage the files you edited rather than using `git add -A` or `git commit -a`.

## Important Notes

- Always source `envsetup.sh` before running build commands
- The UI build generates compressed files in `ui/build`; these are committed by
  the user before a release, not with code changes (see Committing above)
- NATS JetStream stores all application data (one stream per boundary/origin
  pair; see ADR-7)
- System supports TLS with certificates via `siot_mkcert` and `siot_run_tls`;
  in production the NATS listeners' certificate comes from one source in
  `server/certs.go` (files, a directory, or ACME through `server/acme.go`)
- Points and node replies use the binary encoding in `data/point.go` and
  `data/node.go`; protocol buffers (`siot_protobuf`) remain only for Sparkplug B
  and file transfer
- Cross-platform support (Linux, macOS, Windows with ARM variants)
- Embedded systems focus - minimal dependencies and binary size optimization
