// Package store implements the SIOT data store and processes messages.
// Currently data is stored in Genji and Influxdb.
// Direct DB access is not provided and all write data goes through NATS,
// thus making it easy to observe any data changes.
package store
