# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Simple IoT is a Go-based IoT platform with an Elm frontend that enables distributed sensor data collection, telemetry, configuration, and device management. The system runs the same application in both cloud and edge instances, automatically synchronizing data between them using NATS messaging.

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
- `store/` - SQLite storage layer
- `frontend/` - Elm-based web UI
- `modbus/` - Modbus protocol implementation
- `network/` - Network management utilities

### Client Architecture
Most functionality is implemented as clients that:
- Subscribe to relevant node changes via NATS
- Process data and implement business logic
- Publish point updates back to the system
- Are managed by the ClientManager system

Common client types: SerialDev, CanBus, Rule, Db, SignalGenerator, Sync, Metrics, Modbus, OneWire, Shelly, Particle, etc.

### Frontend Architecture
- **Elm SPA**: Single-page application using elm-spa framework
- **Components**: Node-specific UI components in `Components/` directory
- **API**: Communication with backend via HTTP and WebSocket
- **Build**: Uses elm-watch for hot reloading during development

## Development Workflow

1. **Setup**: `source envsetup.sh && siot_setup`
2. **Development**: `siot_watch` (starts hot reloading for both frontend and backend)
3. **Testing**: `siot_test` before submitting changes
4. **Code Quality**: All code must pass `golangci-lint run` and `elm-review`

## Important Notes

- Always source `envsetup.sh` before running build commands
- Frontend build generates compressed `elm.js.gz` file
- SQLite database stores all application data
- System supports TLS with certificates via `siot_mkcert` and `siot_run_tls`
- Protocol buffers used for efficient data serialization (`siot_protobuf`)
- Cross-platform support (Linux, macOS, Windows with ARM variants)
- Embedded systems focus - minimal dependencies and binary size optimization