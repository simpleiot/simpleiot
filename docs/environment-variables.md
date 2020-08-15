---
id: envvars
title: Environment Variables
sidebar_label: Environment Variables
---

Environment variables are used to control various aspects of the application.
The following are currently defined:

- General
  - `SIOT_HTTP_PORT`: http network port the SIOT server attaches to (default
    is 8080)
  - `SIOT_DATA`: directory where any data is stored
  - `SIOT_AUTH_TOKEN`: auth token used for NATS (and eventually HTTP device
    API), default is blank (no auth)
- NATS configuration
  - `SIOT_NATS_PORT`: Port to run NATS on (default is 4222 if not set)
  - `SIOT_NATS_HTTP_PORT`: Port to run NATS monitoring interface (default is
  - `SIOT_NATS_SERVER`: defaults to nats://localhost:4222
  - `SIOT_NATS_TLS_CERT`: points to TLS certificate file. If not set, TLS is not
    used.
  - `SIOT_NATS_TLS_KEY`: points to TLS certificate key
- Particle.io
  - `SIOT_PARTICLE_API_KEY`: key used to fetch data from Particle.io devices
    running [Simple IoT firmware](https://github.com/simpleiot/firmware)
- InfluxDB 1.x
  - `SIOT_INFLUX_URL`: url for influxdb. The presense of this variable enables
    influxdb 1.x support. Typically this is `http://localhost:8086`.
  - `SIOT_INFLUX_USER`: user name for influxdb
  - `SIOT_INFLUX_PASS`: password for influxdb
- Twilio (used for SMS notifications)
  - `TWILIO_SID`: Twilio account SID
  - `TWILIO_AUTH_TOKEN`: Twilio account auth token
  - `TWILIO_FROM`: sending phone number for SMS messages -- must match the phone
    number in the Twilio account.
