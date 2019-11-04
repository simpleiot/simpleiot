# Simple IoT Documentation

## API

The REST API used by the frontend and devices is documented
[here](https://htmlpreview.github.io/?https://github.com/simpleiot/simpleiot/blob/master/docs/api.html) using
[API Blueprint](api.apibp).

### Examples of looking at API data

- install `wget` and `jq`
- `wget -qO - http://localhost:8080/v1/devices | jq -C`

## Environment Variables

Environment variables are used to control various aspects of the application. The
following are currently defined:

- `SIOT_PORT`: network port the SIOT server attaches to
- `SIOT_DATA`: directory where any data is stored
- `SIOT_PARTICLE_API_KEY`: key used to fetch data from Particle.io devices
