# Graphing

Simple IoT is designed to work with several other applications for storing time
series data and viewing this data in graphs.

## InfluxDB

[InfluxDB](https://www.influxdata.com/products/influxdb-overview/) is currently
the recommended way to store historical data. This database is efficient and can
run on embedded platforms like the Raspberry PI as well as desktop and server
machines. To connect SIOT to InfluxDB, simply add an InfluxDB node in your
setup, and fill in the parameters.

## Grafana

[Grafana](https://grafana.com/) is a very powerful graphing solution that works
well with InfluxDB. Although InfluxDB has its own web interface and graphing
capability, generally we find Grafana to be more full featured and easier to
use.
