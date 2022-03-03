# Configuration 

Environment variables are used to control various aspects of the application.
The following are currently defined:

- **General**
  - `SIOT_HTTP_PORT`: http network port the SIOT server attaches to (default
    is 8080)
  - `SIOT_DATA`: directory where any data is stored
  - `SIOT_AUTH_TOKEN`: auth token used for NATS and HTTP device API, default is
    blank (no auth)
- **NATS configuration**
  - `SIOT_NATS_PORT`: Port to run NATS on (default is 4222 if not set)
  - `SIOT_NATS_HTTP_PORT`: Port to run NATS monitoring interface (default
    is 8222)
  - `SIOT_NATS_SERVER`: defaults to nats://localhost:4222
  - `SIOT_NATS_TLS_CERT`: points to TLS certificate file. If not set, TLS is not
    used.
  - `SIOT_NATS_TLS_KEY`: points to TLS certificate key
  - `SIOT_NATS_TLS_TIMEOUT`: Configure the TLS upgrade timeout. NATS defaults to
    a 0.5s timeout for TLS upgrade, but that is too short for some embedded
    systems that run on low end CPUs connected over cellular modems (we've see
    this process take as long as 4s). See NATS
    [documentation](https://docs.nats.io/nats-server/configuration/securing_nats/tls#tls-timeout)
    for more information.
  - `SIOT_NATS_WS_PORT`: Port to run NATS websocket (default is 9222, set to 0
    to disable)
- **Particle.io**
  - `SIOT_PARTICLE_API_KEY`: key used to fetch data from Particle.io devices
    running [Simple IoT firmware](https://github.com/simpleiot/firmware)
