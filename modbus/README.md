# Simple IoT Modbus

This Simple IoT modbus packet is a package that implements both Modbus client
and server functionality. This is a work in progress and currently only supports
Modbus RTU, but can easily be extended for TCP and ASCII.

See [this test](./rtu-end-to-end_test.go) for an example of how to use this
library. Substitute the wire simulator with real serial ports. There are also
standalone
[client](https://github.com/simpleiot/simpleiot/blob/master/cmd/modbus-client/main.go)
and
[server](https://github.com/simpleiot/simpleiot/blob/master/cmd/modbus-server/main.go)
examples.

## Why?

- really want to be able to pass in my own io.ReadWriter into these libs. New
  and better serial libs are continually coming out, and it seems that
  hardcoding serial operations in the modbus lib makes this brittle.
- allows use of https://pkg.go.dev/github.com/simpleiot/simpleiot/respreader to
  do packet framing. This may not be the most efficient, but is super easy and
  eliminates the need to parse packets as they come in.
- passing in io.ReadWriter allows us to easily unit test end to end
  communication (see end-to-end test above)
- library in constructed in a modular fashion from lowest units (PDU) on up, so
  it is easy to test
- not satisfied with the mbserver way of hooking in register operations. Seems
  over complicated, and unfinished.
- want one library for both client/server. This is necessary to test everything,
  and there is no big reason not to do this.
- need a library that is maintained, accepts PRs, etc.
- could not find any good tools for testing modbus (client or server). Most are
  old apps for windows (I use Linux) and are difficult to use. After messing
  around too long in a windows VM, I decided its time for a decent command line
  app that does this. Seems like there should be a simple command line tool that
  can be made to do this and run on any OS (Go is perfect for building this type
  of thing).
