---
id: envvars
title: Environment Variables
sidebar_label: Environment Variables
---

Environment variables are used to control various aspects of the application.
The following are currently defined:

- `SIOT_PORT`: network port the SIOT server attaches to
- `SIOT_DATA`: directory where any data is stored
- `SIOT_PARTICLE_API_KEY`: key used to fetch data from Particle.io devices
- `SIOT_INFLUX_URL`: url for influxdb. The presense of this variable enables
  influxdb 1.x support. Typically this is `http://localhost:8086`.
- `SIOT_INFLUX_USER`: user name for influxdb
- `SIOT_INFLUX_PASS`: password for influxdb
