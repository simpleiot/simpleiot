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

NOTE, it is important to set an auth token - otherwise there is no restriction
on accessing the device API.

## NATS

Currently devices communicating via NATS use a common auth token. This is the
main limitation of the current model: one token protects the whole fleet, so
revoking a single device means rotating the token everywhere.

### Per-device credentials (planned)

The planned replacement issues each device its own credential and scopes it to
that device's data, so access can be revoked per device. See
[Per-device credentials](../user/sync.md#per-device-credentials-planned) for the
behavior users will see. The intended implementation:

- Device credentials are nodes in the upstream's tree (a credential node under
  the device, or a user-style node with a hashed secret), so issuing and
  revoking them is an ordinary node edit and is synced and exported like any
  other configuration.
- Authentication is performed by the embedded NATS server through
  [auth callout](https://docs.nats.io/running-a-nats-service/configuration/securing_nats/auth_callout)
  (or custom client authentication for the embedded server), which lets Simple
  IoT look the credential up in its own store at connect time rather than
  maintaining a separate NATS accounts file.
- Permissions returned at connect time restrict the device to the subjects and
  JetStream streams of its own boundary. The stream-per-boundary layout
  ([ADR-7](adr/7-jetstream-store.md)) is what makes a one-rule-per-device grant
  possible.
- Revocation disconnects any live session using that credential and rejects
  reconnects.
- The shared auth token keeps working for a transition period and for
  deployments that do not need per-device isolation.
- The same authorizer covers the built-in MQTT broker, so an MQTT gateway can be
  limited to its own topic prefix.

Long term we plan to leverage the NATS
[security model](https://docs.nats.io/nats-concepts/security) for user and
device authn/authz.:

- [NATS authentication](https://docs.nats.io/running-a-nats-service/configuration/securing_nats/auth_intro)
- [NATS authorization](https://docs.nats.io/running-a-nats-service/configuration/securing_nats/authorization)
