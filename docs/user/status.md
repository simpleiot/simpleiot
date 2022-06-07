# Status

The Simple IoT project is still in a heavy development phase. Most of the core
concepts are stable, but APIs, packet formats, and implementation will continue
to change for some time yet. SIOT has been used in several production systems to
date with good success, but be prepared to work with us (report issues, help fix
bugs, etc.) if you want to use it now.

## Database

We are currently using a key/value store based on bbolt as the data store. This
will likely be
[changed out for SQLite](https://github.com/simpleiot/simpleiot/issues/320) in
the near future. That ability to run SQLite in a pure Go application only
recently because available. This should not affect developers much as all data
access goes through NATS, but data for existing instances will need to be
manually migrated from the old to the new disk formats. This is done by
exporting the data using the old version, and then importing the data using the
new version. SQLite has a very stable disk format so we don't anticipate any
migration issues in the future after we switch to SQLite.

## Handling of high rate sensor data

Currently each point change requires quite a bit computation to update the HASH
values in upstream graph nodes. For repetitive data, this is not necessary as
new values are continually coming in, so we will at some point make an option to
specify points values as repetitive. This will allow SIOT to scale to more
devices and higher rate data.

## User Interface

The web UI is currently polling the SIOT backend every 4 seconds via HTTP. This
works OK for small data sets, but uses more data than necessary and has a
latency of up to 4s. Long term we will run a
[NATS client](https://github.com/simpleiot/simpleiot/tree/master/frontend/lib)
in the frontend over a websocket so the UI response is real-time and new data
gets pushed to the browser.

# Errata

Any issues we find during testing we log in
[Github issues](https://github.com/simpleiot/simpleiot/issues), so if you
encounter something unexpected, please search issues first. Feel free to add
your observations and let us know if an issues is impacting you. Several issues
to be aware of:

- we don't [handle loops](https://github.com/simpleiot/simpleiot/issues/294) in
  the graph tree yet. This will render the instance unusable and you'll have to
  clean the database and start over.
- for now, create a different email for the admin user on each instance.
  [See #366](https://github.com/simpleiot/simpleiot/issues/366).
- there are still several corner cases with upstream connections that need
  improved. (#367, #366, #339)
