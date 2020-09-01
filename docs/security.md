---
id: security
title: Security
sidebar_label: Security
---

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

## NATS

Currently devices communicating via NATS use a common auth token. It would be
nice to move to something where each device has its own authentication (TODO,
explore NATS adavanced auth options).
