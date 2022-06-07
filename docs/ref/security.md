# Security

Several classes of users/devices will need access to the system:

- People (Web UI, currently HTTP)
  - Site admin
  - Device admin
  - Device viewer
- Devices (NATS, HTTP)

## HTTP

The Web UI uses JWT (JSON web tokens).

Devices can also communicate via HTTP and use a simple auth token. Eventually
may want to switch to JWT or something similiar to what NATS uses.

NOTE, it is important to set an auth token -- otherwise there is no restriction
on accessing the device API.

## NATS

Currently devices communicating via NATS use a common auth token. It would be
nice to move to something where each device has its own authentication (TODO,
explore NATS adavanced auth options).

Long term we plan to leverage the NATS
[security model](https://docs.nats.io/nats-concepts/security) for user and
device authn/authz.:

- [NATS authentication](https://docs.nats.io/running-a-nats-service/configuration/securing_nats/auth_intro)
- [NATS authorization](https://docs.nats.io/running-a-nats-service/configuration/securing_nats/authorization)
