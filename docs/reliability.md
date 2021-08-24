+++
title = "Reliability"
weight = 5
+++

Reliability is an important consideration in any IoT system as these systems are
often used to monitor and control critical systems and processes. Performance is
a key aspect of reliability because if the system is not performing well, then
it can't keep up and do its job.

## Point Metrics

The fundamental operation of SimpleIoT is that it process `points`, which are
changes to `nodes`. If the system can't process points at the rate they are
coming in, then we have a problem as data will start to back up and the system
will not be responsive.

Points and other data flow through the NATS messaging system, therefore it is
perhaps the first place to look. We track several metrics that are written to
the root device node to help track how the system is performing.

The NATS client buffers messages that are received for each subscription and
then messages are
[dispatched serially one message at a time](https://docs.nats.io/developing-with-nats/receiving/async).
If the application can't keep up with processing messages, then the number of
buffered messages increases. This number is occasionally read and then
min/max/avg writen to the `metricNatsPending*` points in the root device node.

The time required to process points is tracked in the `metricNatsCycle*` points
in the root device node. The cycle time is in milliseconds.

We also track point throughput (messages/sec) for various NATS subjects in the
`metricNatsThroughput*` points.

These metrics should be graphed and notifications sent when they are out of the
normal range. Rules that trigger on the point type can be installed high in the
tree above a group of devices so you don't have to write rules for every device.

## Database interactions

Database operations greatly affect system performance. When Points come into the
system, we need to store this data in the primary (ex Genji) and time series
stores (ex InfluxDB). The time it takes to read and write data greatly impacts
how much data we can handle.

## IO failures

All errors reading/writing IO devices should be tracked at both the device and
bus level. These can be observed over time and abnormal rates can trigger
notifications. Error counts should be reported at a low rate to avoid using
bandwidth and resources -- especially if multiple counts are incremented on an
error (IO and bus).

## Logging

Many errors are currently reported as log messages. Eventually some effort
should be made to turn these into error counts and possibly store them in the
time series store for later analysis.
