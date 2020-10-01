# Research

This document contains information that has been researched during the course of
creating Simple IoT.

## Synchronization

An IoT system is inherently distributed. At a minimum, there are three
components:

1. device (Go, C, etc.)
1. cloud (Go)
1. multiple browsers (Elm, Js)

Data can be changed in any of the above locations and must be seamlessly
synchronized to other locations. Failing to consider this simple requirement
early in building the system can make for brittle and overly complex systems.

### Resgate

[resgate.io](https://resgate.io) is an interesting project that solves the
problem of creating a real-time API gateway where web clients are synchronized
seamlessly. This project uses NATS.io for a backbone, which makes it interesting
as NATS is core to this project.

The Resgate system is primarily concerned with synchronizing browser contents.

### couch/pouchdb
