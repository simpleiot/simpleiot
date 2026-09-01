# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with
code in this repository.

## Project Overview

Simple IoT is a Go-based IoT platform with an Elm frontend that enables
distributed sensor data collection, telemetry, configuration, and device
management. The system runs the same application in both cloud and edge
instances, automatically synchronizing data between them using NATS messaging.

## Build System and Common Commands

### Setup and Dependencies

```bash
# Source environment setup (required for all operations)
source envsetup.sh

# Initial setup - installs frontend deps and configures Elm
siot_setup

# Install frontend dependencies only
siot_install_frontend_deps
```

### Building

```bash
# Build everything (frontend + backend)
siot_build

# Build frontend only (Elm SPA)
siot_build_frontend

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
go test -race ./...
```

### Linting and Code Quality

```bash
# Backend linting (uses golangci-lint with revive and goimports)
golangci-lint run

# Frontend linting and review
cd frontend && npx elm-review
cd frontend && npx elm-test
```

## Architecture

### Core Concepts

- **Nodes**: Core data structures containing arrays of Points
- **Points**: Individual data values with timestamps and metadata
- **Graph Structure**: Data organized as a DAG (directed acyclic graph)
- **Clients**: Independent components that implement specific functionality
- **NATS Messaging**: All data flows through embedded NATS message bus

### Key Directories

- `cmd/siot/` - Main application entry point
- `server/` - Server core functionality and HTTP API
- `client/` - Client implementations (most functionality lives here)
- `api/` - HTTP API handlers and routing
- `data/` - Core data structures (Node, Point, etc.)
- `store/` - JetStream storage layer (boundary-origin streams, see ADR-7)
- `frontend/` - Elm-based web UI
- `modbus/` - Modbus protocol implementation
- `network/` - Network management utilities

### Client Architecture

Most functionality is implemented as clients that:

- Subscribe to relevant node changes via NATS
- Process data and implement business logic
- Publish point updates back to the system
- Are managed by the ClientManager system

Common client types: SerialDev, CanBus, Rule, Db, SignalGenerator, Sync,
Metrics, Modbus, OneWire, GPIO, Mqtt, Shelly, Particle, User, MsgService, etc.

A node can sit in several places in the tree, and a client runs on the node's
primary edge and on edges with no role, never on a mirror. Adding a client means
classifying its node type in `primaryTypes` or `treeScopedTypes` in
`data/edge_role.go`; a test fails if it is in neither. See
[Primary and mirror edges](docs/ref/data.md#primary-and-mirror-edges).

### Frontend Architecture

- **Elm SPA**: Single-page application using elm-spa framework
- **Components**: Node-specific UI components in `Components/` directory
- **API**: Communication with backend via HTTP and WebSocket
- **Build**: Uses elm-watch for hot reloading during development

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
4. **Code Quality**: All code must pass `golangci-lint run` and `elm-review`

## Plans

Implementation plans are stored in the `plans/` directory. See `plans/plans.md`
for an index of all plans and their status.

When working through a plan, commit after each phase completes. Update the
changelog (`CHANGELOG.md`), `CLAUDE.md`, and any relevant documentation as part
of each phase.

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

Leave `frontend/public/dist/elm.js.gz` out of commits. The frontend build
regenerates it constantly, so it shows up as modified during normal work. It is
committed only right before a release, in its own commit. When staging changes,
stage the files you edited rather than using `git add -A` or `git commit -a`,
and if the built artifact does get committed by mistake, drop it with
`git restore --source=HEAD~1 --staged frontend/public/dist/elm.js.gz` followed
by `git commit --amend`.

## Important Notes

- Always source `envsetup.sh` before running build commands
- Frontend build generates compressed `elm.js.gz` file (see Committing above —
  it is not committed with regular changes)
- NATS JetStream stores all application data (one stream per boundary/origin
  pair; see ADR-7)
- System supports TLS with certificates via `siot_mkcert` and `siot_run_tls`
- Points and node replies use the binary encoding in `data/point.go` and
  `data/node.go`; protocol buffers (`siot_protobuf`) remain only for Sparkplug B
  and file transfer
- Cross-platform support (Linux, macOS, Windows with ARM variants)
- Embedded systems focus - minimal dependencies and binary size optimization
