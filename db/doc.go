// Package db implements database store code -- currently Genji and Influxdb. Also contains NATS
// handlers to receive data. This allows us to keep the db write functions private and force
// all write data through NATS, and thus makes it easy to observe any data changes.
package db
