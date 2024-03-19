# Database Client

The main [SIOT store](../ref/store.md) is SQLite. SIOT supports additional
database clients for purposes such as storing time-series data.

## InfluxDB 2.x

## Victoria Metrics

Victoria Metrics
[supports the InfluxDB v2](https://docs.victoriametrics.com/#how-to-send-data-in-influxdb-v2-format)
line protocol; therefore, it can be used for numerical data. Victoria Metrics
[does not support storing strings](https://stackoverflow.com/questions/66406899/does-victoriametrics-have-some-way-to-store-string-value-instead-float64).
