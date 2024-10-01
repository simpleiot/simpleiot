# Status

The Simple IoT project is still in a heavy development phase. Most of the core
concepts are stable, but APIs, packet formats, and implementation will continue
to change for some time yet. SIOT has been used in several production systems to
date with good success, but be prepared to work with us (report issues, help fix
bugs, etc.) if you want to use it now.

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

## Security

Currently, and device that has access to the system can write or write to any
data in the system. This may be adequate for small or closed systems, but for
larger systems, we need per-device authn/authz. See
[issue #268](https://github.com/simpleiot/simpleiot/issues/268),
[PR #283](https://github.com/simpleiot/simpleiot/pull/283), and our
[security document](../ref/security.md) for more information.

## Errata

Any issues we find during testing we log in
[Github issues](https://github.com/simpleiot/simpleiot/issues), so if you
encounter something unexpected, please search issues first. Feel free to add
your observations and let us know if an issues is impacting you. Several issues
to be aware of:

- we don't [handle loops](https://github.com/simpleiot/simpleiot/issues/294) in
  the graph tree yet. This will render the instance unusable and you'll have to
  clean the database and start over.
