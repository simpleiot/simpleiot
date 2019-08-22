# Simple IoT Documentation

## API

The REST API used by the frontend and devices is documented
[here](https://htmlpreview.github.io/?https://github.com/simpleiot/simpleiot/blob/master/docs/api.html) using
[API Blueprint](api.apibp).

### Examples of looking at API data

- install `wget` and `jq`
- `wget -qO - http://localhost:8080/v1/devices | jq -C`
