# Security

Users and downstream devices will need access to a Simple IoT instance. Simple
IoT currently provides access via HTTP and NATS.

## Server

For cloud/server deployments, we recommend installing a web server like Caddy in
front of Simple IoT. See the [Installation page](../user/installation.md) for
more information.

## Edge

Simple IoT Edge instances initiate all connections to upstream instances;
therefore, no incoming connections are required on edge instances and all
incoming ports can be firewalled.

## HTTP

The Web UI uses JWT (JSON web tokens).

Devices can also communicate via HTTP and use a simple auth token. Eventually
may want to switch to JWT or something similar to what NATS uses.

NOTE, it is important to set an auth token -- otherwise there is no restriction
on accessing the device API.

## NATS

Currently devices communicating via NATS use a common auth token. It would be
nice to move to something where each device has its own authentication (TODO,
explore NATS advanced auth options).

Long term we plan to leverage the NATS
[security model](https://docs.nats.io/nats-concepts/security) for user and
device authn/authz.:

- [NATS authentication](https://docs.nats.io/running-a-nats-service/configuration/securing_nats/auth_intro)
- [NATS authorization](https://docs.nats.io/running-a-nats-service/configuration/securing_nats/authorization)
